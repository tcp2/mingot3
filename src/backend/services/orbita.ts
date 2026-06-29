import { spawn } from "node:child_process";
import {
	closeSync,
	existsSync,
	globSync,
	mkdirSync,
	openSync,
	readFileSync,
	rmSync,
	writeFileSync,
} from "node:fs";
import net from "node:net";
import { homedir } from "node:os";
import { join, resolve } from "node:path";
import { parse } from "node:querystring";
import WebSocket from "ws";
import { monotonicFactory } from "ulid";
import { getTokens } from "../../../scripts/get-gologin-token.mjs";
import { freeSlot, getGridConfig } from "./gridTracker";
import { sanitizeProfile } from "./backup";
import { profileBus } from "../events";

// ─── UTILS ────────────────────────────────────────────────────────────────────
// ULID is similar to UUID but sortable by time. monotonicFactory ensures IDs generated in the same ms sort correctly.
const ulid = monotonicFactory();

async function waitForDebugPort(port, maxWaitMs = 4000) {
	const start = Date.now();
	while (Date.now() - start < maxWaitMs) {
		try {
			const controller = new AbortController();
			const timeoutId = setTimeout(() => controller.abort(), 200);
			const res = await fetch(`http://127.0.0.1:${port}/json/version`, {
				signal: controller.signal,
			});
			clearTimeout(timeoutId);
			if (res.ok) return true;
		} catch (_err) {
			// Connection refused or timeout
		}
		await new Promise((r) => setTimeout(r, 200));
	}
	return false;
}

async function getFreePort(): Promise<number> {
	return new Promise((resolve, reject) => {
		const server = net.createServer();
		server.unref();
		server.on("error", reject);
		server.listen(0, "127.0.0.1", () => {
			const port = (server.address() as any).port;
			server.close(() => resolve(port));
		});
	});
}

// ─── CONFIGURATION ────────────────────────────────────────────────────────────
// In Electron production, APP_ROOT is set to app.getPath('userData') so profiles
// are stored in %APPDATA%/Mingot/ instead of next to the exe.
const PROJECT_ROOT = process.env.APP_ROOT || process.cwd();

const BROWSER_EXE = join(
	homedir(),
	".gologin",
	"browser",
	"orbita-browser-149",
	"chrome.exe",
);
const GOLOGIN_DIR = join(PROJECT_ROOT, "profiles");
const ZERO_FILE = join(PROJECT_ROOT, "config", "zero_profile.json");

const TZ_URL = "https://geo.myip.link";

const SECURED_ORBITA_OPTS = [
	"webGpu",
	"webgl",
	"webglParams",
	"webRTC",
	"webrtc",
	"mediaDevices",
	"plugins",
	"audioContext",
	"canvasMode",
	"canvasNoise",
	"webgl_noice_enable",
	"webglNoiceEnable",
	"webgl_noise_enable",
	"client_rects_noise_enable",
	"webgl_noise_value",
	"webglNoiseValue",
	"getClientRectsNoice",
	"get_client_rects_noise",
];

// ─── API HELPER ───────────────────────────────────────────────────────────────
class Api {
	static async request(method, url, headers = {}, data = null) {
		const fetchOptions: any = {
			method,
			headers: { ...headers },
		};

		if (data) {
			fetchOptions.headers["Content-Type"] = "application/json";
			fetchOptions.body = data;
		}

		const response = await fetch(url, fetchOptions);

		console.log(`[API] ${response.status} ${method} ${url}`);
		const bodyText = await response.text();
		if (!response.ok) {
			throw new Error(`HTTP ${response.status}: ${bodyText.slice(0, 300)}`);
		}

		try {
			return JSON.parse(bodyText);
		} catch {
			return bodyText;
		}
	}

	static async get(url, headers = {}) {
		return Api.request("GET", url, headers);
	}

	static fetchLocalApiTokens() {
		try {
			const rawTokens = getTokens();
			const tokens = [];
			for (const { token, payload } of rawTokens) {
				if (token && payload?.sub) {
					tokens.push({ token, userId: payload.sub });
				}
			}
			return tokens;
		} catch (err) {
			console.warn(`[WARN] Failed to fetch API_TOKENs: ${err.message}`);
			return [];
		}
	}

	/**
	 * Lấy timezone IP hiện tại để cấu hình trình duyệt (phục vụ ẩn danh)
	 */
	static async fetchTimezone() {
		return await Api.get(TZ_URL);
	}

	/**
	 * Lấy Profile Token từ GoLogin để cấp quyền cho Orbita Browser
	 * @param profileId ID của profile trên hệ thống GoLogin
	 * @param apiToken Token uỷ quyền (Bearer token)
	 */
	static async fetchProfileToken(profileId, apiToken) {
		if (!apiToken)
			throw new Error("API_TOKEN is empty. Cannot fetch profile token.");
		const res = await Api.get(
			`https://api.gologin.com/browser/features/${profileId}/profile-params-for-orbita-token`,
			{ Authorization: `Bearer ${apiToken}`, "User-Agent": "Selenium-API" },
		);
		if (!res.token)
			throw new Error(`API returned empty token: ${JSON.stringify(res)}`);
		return res.token;
	}

	/**
	 * Tải Fingerprint (dấu vân tay trình duyệt) của profile từ GoLogin.
	 * Gồm các cấu hình Canvas, WebGL, Fonts, v.v.
	 */
	static async fetchProfileFingerprint(profileId, apiToken) {
		if (!apiToken) throw new Error("API_TOKEN is empty.");
		return await Api.get(`https://api.gologin.com/browser/${profileId}`, {
			Authorization: `Bearer ${apiToken}`,
			"User-Agent":
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36 ",
		});
	}

	/**
	 * Cập nhật/Tạo mới fingerprint (như hệ điều hành giả lập, v.v.)
	 */
	static async refreshProfileFingerprint(profileId, apiToken) {
		if (!apiToken) throw new Error("API_TOKEN is empty.");
		return await Api.get(
			`https://api.gologin.com/browser/fingerprint?os=win&osSpec=win11&template=${profileId}`,
			{
				Authorization: `Bearer ${apiToken}`,
				"User-Agent":
					"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
			},
		);
	}
	/**
	 * Lấy ID của profile đầu tiên trong tài khoản GoLogin.
	 * Thường dùng làm profile "mẫu" (template) khi cần fetch dữ liệu mặc định.
	 */
	static async fetchFirstProfileId(apiToken) {
		if (!apiToken) throw new Error("API_TOKEN is empty.");
		const res = await Api.get(`https://api.gologin.com/browser/v2`, {
			Authorization: `Bearer ${apiToken}`,
			"User-Agent":
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
		});
		if (res?.profiles && res.profiles.length > 0) {
			return res.profiles[0].id;
		}
		throw new Error("No profiles found in GoLogin account.");
	}
}

// ─── CONFIG BUILDER ───────────────────────────────────────────────────────────
class ConfigBuilder {
	static generateProfileName() {
		// Đếm tổng số thư mục profile hiện có (bắt đầu bằng MG-)
		const dirs = globSync("MG-*", {
			cwd: GOLOGIN_DIR,
			onlyDirectories: true,
		} as any);
		const nextNum = dirs.length + 1;

		return `P${nextNum.toString().padStart(3, "0")}`;
	}

	static buildIntl(profile, tz) {
		const lang = (profile.navigator?.language ?? "en-US").split(",")[0];
		const autoLang = profile.autoLang ?? true;

		if (!autoLang || !tz.languages) {
			return {
				accept_languages: lang,
				selected_languages: lang,
				app_locale: lang.split("-")[0],
				forced_languages: [lang.split("-")[0]],
			};
		}

		const firstLocale = tz.languages.split(",")[0];
		const tzLang = tz.country ? `${firstLocale}-${tz.country}` : firstLocale;
		const langCode = tzLang.split("-")[0];
		const result = [...new Set([tzLang, langCode, "en-US", "en"])];

		return {
			accept_languages: result.join(","),
			selected_languages: result.join(","),
			app_locale: langCode,
			forced_languages: [langCode],
		};
	}

	static buildOrbita(profile, tz, gologinPrefs, profileToken) {
		const intl = ConfigBuilder.buildIntl(profile, tz);
		const clientOpts = {};
		for (const key of SECURED_ORBITA_OPTS) {
			if (key in gologinPrefs) clientOpts[key] = gologinPrefs[key];
		}
		return {
			intl,
			gologin: Object.assign({ profile_token: profileToken }, clientOpts),
		};
	}

	static buildPrefs(profile, tz, userId, localSessionId, profileName) {
		const nav = profile.navigator ?? {};
		const metadata = profile.webGLMetadata ?? {};
		const webGL = profile.webGL ?? {};
		const clientRects = profile.clientRects ?? {};
		const canvas = profile.canvas ?? {};
		const [w, h] = (nav.resolution ?? "1920x1080").split("x");
		const langHeader = nav.language ?? "";
		const parts = (profile.startUrl ?? "").split(",");
		const noise = (webGL.mode ?? "") === "noise";

		return {
			_id: localSessionId || "",
			profile_id: profile.id ?? "",
			name: profileName || profile.name || "",
			userId: userId,
			userPlanName: "Forever Free",
			is_m1:
				(profile.os ?? "") === "mac" && (profile.osSpec ?? "").includes("M"),

			dns: profile.dns ?? [],
			proxy: {
				username: profile.proxy?.username ?? "",
				password: profile.proxy?.password ?? "",
			},
			webRTC: profile.webRTC ?? {},
			webglParams: profile.webglParams ?? [],
			webGpu: profile.webGpu ?? [],
			navigator: {
				platform: nav.platform ?? "",
				max_touch_points: nav.maxTouchPoints ?? 0,
			},
			userAgent: nav.userAgent ?? "",
			screenWidth: parseInt(w, 10),
			screenHeight: parseInt(h, 10),
			doNotTrack: nav.doNotTrack ?? false,
			hardwareConcurrency: nav.hardwareConcurrency ?? 2,
			deviceMemory: (nav.deviceMemory ?? 2) * 1024,
			languages: langHeader.split(",")[0] || "en-US",
			langHeader,
			mobile: {
				enable: (profile.os ?? "") === "android",
				width: profile.screenWidth ?? 1920,
				height: profile.screenHeight ?? 1080,
				device_scale_factor: profile.devicePixelRatio ?? 1,
			},
			webGl: {
				vendor: metadata.vendor ?? "",
				renderer: metadata.renderer ?? "",
				mode: (metadata.mode ?? "") === "mask",
			},
			webgl: {
				metadata: {
					vendor: metadata.vendor ?? "",
					renderer: metadata.renderer ?? "",
					mode: (metadata.mode ?? "") === "mask",
				},
			},
			webgl_noice_enable: noise,
			webglNoiceEnable: noise,
			webgl_noise_enable: noise,
			webgl_noise_value: webGL.noise ?? null,
			webglNoiseValue: webGL.noise ?? null,
			getClientRectsNoice:
				clientRects.noise ?? webGL.getClientRectsNoise ?? null,
			client_rects_noise_enable: (clientRects.mode ?? "") === "noise",
			media_devices: {
				enable: profile.mediaDevices?.enableMasking ?? true,
				uid: profile.mediaDevices?.uid ?? "",
				audioInputs: profile.mediaDevices?.audioInputs ?? 1,
				audioOutputs: profile.mediaDevices?.audioOutputs ?? 1,
				videoInputs: profile.mediaDevices?.videoInputs ?? 1,
			},
			plugins: {
				all_enable: profile.plugins?.enableVulnerable ?? true,
				flash_enable: profile.plugins?.enableFlash ?? true,
			},
			storage: { enable: profile.storage?.local ?? true },
			audioContext: {
				enable: (profile.audioContext?.mode ?? "off") !== "off",
				noiseValue: profile.audioContext?.noise ?? "",
			},
			canvas: { mode: canvas.mode ?? "" },
			canvasMode: canvas.mode ?? "",
			canvasNoise: canvas.noise ?? "",
			startupUrl: (parts[0] ?? "").trim(),
			startup_urls: parts.map((s) => s.trim()).filter(Boolean),
			geolocation: {
				mode: profile.geolocation?.mode ?? "prompt",
				latitude: parseFloat(tz.ll?.[0] ?? 0),
				longitude: parseFloat(tz.ll?.[1] ?? 0),
				accuracy: parseFloat(tz.accuracy ?? 0),
			},
			timezone: { id: tz.timezone ?? "" },
		};
	}

	static getMajorVersion(userAgent) {
		const m = (userAgent || "").match(/Chrome\/(\d+)\./);
		return m ? parseInt(m[1], 10) : 0;
	}
}

// ─── ORBITA LAUNCHER ──────────────────────────────────────────────────────────
class OrbitaLauncher {
	existingSessionId: string | null;
	profileName: string;
	debugPort: number;
	localSessionId!: string;
	profilePath!: string;
	localAuths: any[];
	userId: string;
	tz: any;
	profile: any;
	profileToken: string;
	resolution: string;
	onlineProfileId?: string;

	constructor(existingSessionId: string | null = null) {
		this.existingSessionId = existingSessionId;
		this.profileName = "";
		this.debugPort = 0; // Will be assigned in init()

		if (existingSessionId) {
			this.localSessionId = existingSessionId;
			this.profilePath = join(GOLOGIN_DIR, this.localSessionId);
			const prefsPath = join(this.profilePath, "Default", "Preferences");
			let prefs: any = {};
			if (existsSync(prefsPath)) {
				prefs = JSON.parse(readFileSync(prefsPath, "utf-8"));
				this.profileName = prefs.gologin?.name || this.localSessionId;
			} else {
				this.profileName = this.localSessionId;
			}
		} else {
			if (!existsSync(GOLOGIN_DIR)) {
				mkdirSync(GOLOGIN_DIR, { recursive: true });
			}

			this.profileName = ConfigBuilder.generateProfileName();

			const randomId = ulid();
			this.localSessionId = `MG-${randomId}`;
			this.profilePath = join(GOLOGIN_DIR, this.localSessionId);
		}

		this.localAuths = Api.fetchLocalApiTokens();
		this.userId = "";
		this.tz = null;
		this.profile = null;
		this.profileToken = "";
		this.resolution = "1920x1080";
	}

	/**
	 * Khởi tạo profile:
	 * 1. Lấy thông tin Timezone.
	 * 2. Quét qua các GoLogin token (từ token_stats.json) để tìm token còn hoạt động.
	 * 3. Fetch fingerprint và profile_token tương ứng để cấp phép cho Orbita.
	 * Quá trình này sẽ lấy dữ liệu từ cloud GoLogin.
	 */
	async init() {
		this.debugPort = await getFreePort();

		if (this.existingSessionId) {
			console.log(
				`\n[INFO] Loading existing offline profile: ${this.existingSessionId}`,
			);
			console.log(`[INFO] Profile Path: ${this.profilePath}`);
			if (!existsSync(this.profilePath)) {
				throw new Error(`Profile directory not found: ${this.profilePath}`);
			}
			// Thử đọc độ phân giải từ cấu hình cũ
			try {
				const prefs = JSON.parse(
					readFileSync(
						join(this.profilePath, "Default", "Preferences"),
						"utf-8",
					),
				);
				if (prefs.gologin?.screenWidth && prefs.gologin.screenHeight) {
					this.resolution = `${prefs.gologin.screenWidth}x${prefs.gologin.screenHeight}`;
				}
			} catch (e: any) {
				console.error("Caught error:", e.message);
			}
			return;
		}

		console.log("\n══════════════════════════════════════════════════");
		console.log("STEP 1: Load Timezone");
		console.log("══════════════════════════════════════════════════");

		if (!existsSync(ZERO_FILE)) {
			throw new Error("Missing zero_profile.json");
		}

		this.tz = await Api.fetchTimezone();
		console.log("Timezone:   ", this.tz.timezone, "| IP:", this.tz.ip);

		console.log("\n══════════════════════════════════════════════════");
		console.log("STEP 2: Fetch Profile Data (Fingerprint & Token)");
		console.log("══════════════════════════════════════════════════");

		let success = false;
		let lastError = null;
		let tokenStats: any = {};
		const tokenStatsFile = join(GOLOGIN_DIR, "token_stats.json");
		if (existsSync(tokenStatsFile)) {
			tokenStats = JSON.parse(readFileSync(tokenStatsFile, "utf8"));
		}

		// Sắp xếp token: ưu tiên những token ít bị lỗi (đặc biệt là lỗi 403) lên trước
		this.localAuths.sort((a, b) => {
			const aErrors = tokenStats[a.token] || 0;
			const bErrors = tokenStats[b.token] || 0;
			return aErrors - bErrors;
		});

		for (const localToken of this.localAuths) {
			console.log(
				`[+] Requesting first profile and new fingerprint via API...`,
			);
			try {
				this.onlineProfileId = await Api.fetchFirstProfileId(localToken.token);
				await Api.refreshProfileFingerprint(
					this.onlineProfileId,
					localToken.token,
				);
				console.log(`[${this.onlineProfileId}] Fetch fingerprint success`);

				const tokenData = await Api.fetchProfileToken(
					this.onlineProfileId,
					localToken.token,
				);
				const fpData = await Api.fetchProfileFingerprint(
					this.onlineProfileId,
					localToken.token,
				);

				this.userId = localToken.userId;
				this.profileToken = tokenData;
				this.profile = fpData;
				this.resolution = this.profile.navigator?.resolution ?? "1920x1080";
				success = true;

				// Reset điểm phạt về 0 khi token thành công
				tokenStats[localToken.token] = 0;
				writeFileSync(tokenStatsFile, JSON.stringify(tokenStats));

				console.log("Online Profile ID:", this.profile.id);
				console.log("Session ID:       ", this.localSessionId);
				console.log("Name:             ", this.profileName);
				console.log("UserAgent:        ", this.profile.navigator?.userAgent);
				console.log(
					`[OK] Fetched fingerprint & token successfully using account ${this.userId}!`,
				);
				success = true;
				break;
			} catch (err) {
				lastError = err;
				console.log(
					`[WARN] Token for ${localToken.userId} failed: ${err.message}`,
				);
				// Cộng điểm phạt cho token bị lỗi
				tokenStats[localToken.token] = (tokenStats[localToken.token] || 0) + 1;
				writeFileSync(tokenStatsFile, JSON.stringify(tokenStats));
			}
		}

		if (!success) {
			throw new Error(
				`Cannot fetch fresh token/fingerprint for any profile. Last error: ${lastError?.message}`,
			);
		}
	}

	/**
	 * Cấu trúc và sinh ra các file cấu hình cần thiết để trình duyệt Orbita (GoLogin)
	 * có thể đọc được (ví dụ: Preferences, orbita.config, v.v.)
	 */
	buildProfile(force = false) {
		if (this.existingSessionId) {
			console.log(`[INFO] Skip building configs for existing profile.`);
			return;
		}

		console.log("\n══════════════════════════════════════════════════");
		console.log("STEP 3: Write Configs & Directories");
		console.log("══════════════════════════════════════════════════");

		if (force && existsSync(this.profilePath)) {
			console.log("[FORCE] Deleting old profile directory...");
			rmSync(this.profilePath, { recursive: true, force: true });
		}

		const defPath = join(this.profilePath, "Default");
		const netPath = join(defPath, "Network");
		if (!existsSync(netPath)) mkdirSync(netPath, { recursive: true });

		try {
			// Zero profile bookmarks
			const zeroData = JSON.parse(readFileSync(ZERO_FILE, "utf-8"));
			writeFileSync(
				join(defPath, "Bookmarks"),
				JSON.stringify(zeroData, null, 2),
			);

			// Empty cookies DBs
			writeFileSync(join(netPath, "Cookies"), "");
			writeFileSync(join(defPath, "Cookies"), "");

			// Build Preferences
			const prefs = ConfigBuilder.buildPrefs(
				this.profile,
				this.tz,
				this.userId,
				this.localSessionId,
				this.profileName,
			);
			(prefs as any).profile_token = this.profileToken; // Cache token to avoid API limits on relaunch

			// Handle proxy in Preferences (for Orbita >= 135)
			const mVer = ConfigBuilder.getMajorVersion(
				this.profile.navigator?.userAgent,
			);
			const proxyData = this.profile.proxy ?? { mode: "none" };
			if (mVer >= 135 && (proxyData.mode ?? "none") !== "none") {
				const auth = proxyData.username
					? `${encodeURIComponent(proxyData.username)}:${encodeURIComponent(proxyData.password)}@`
					: "";
				zeroData.proxy = {
					mode: "fixed_servers",
					schema: proxyData.mode ?? "",
					username: encodeURIComponent(proxyData.username ?? ""),
					password: encodeURIComponent(proxyData.password ?? ""),
					server: `${proxyData.mode ?? ""}://${auth}${proxyData.host ?? ""}:${proxyData.port ?? 0}`,
				};
			}

			zeroData.gologin = prefs;
			writeFileSync(
				join(defPath, "Preferences"),
				JSON.stringify(zeroData, null, 2),
			);
			console.log("[OK] Written Preferences");

			// Build orbita.config
			const orbitaConfig = ConfigBuilder.buildOrbita(
				this.profile,
				this.tz,
				prefs,
				this.profileToken,
			);
			writeFileSync(
				join(this.profilePath, "orbita.config"),
				JSON.stringify(orbitaConfig, null, "\t"),
				"utf-8",
			);
			console.log("[OK] Written orbita.config");
		} catch (err) {
			if (err.code === "EBUSY" || err.code === "EPERM") {
				console.warn(
					`\n[WARN] Profile đang được mở (File bị khoá)! Bỏ qua bước ghi đè cấu hình...`,
				);
			} else {
				throw err;
			}
		}

		console.log(`\n[OK] Profile ready at ${this.profilePath}`);
	}

	/**
	 * Thực thi (spawn) trình duyệt dựa trên các thông số đã khởi tạo.
	 * Mở port debug và lắng nghe sự kiện đóng để dọn dẹp port.
	 */
	async launch(gridConfig?: { x: number; y: number; w: number; h: number }) {
		console.log("\n══════════════════════════════════════════════════");
		console.log("STEP 4: Launch Browser");
		console.log("══════════════════════════════════════════════════");

		if (!existsSync(BROWSER_EXE)) {
			throw new Error(`Browser not found: ${BROWSER_EXE}`);
		}

		// Chặn mở trùng bằng cách check file lock của Chromium (file Cookies)
		const cookiePath = join(this.profilePath, "Default", "Network", "Cookies");
		if (existsSync(cookiePath)) {
			try {
				const fd = openSync(cookiePath, "r+");
				closeSync(fd);
			} catch (err) {
				if (err.code === "EBUSY" || err.code === "EPERM") {
					throw new Error(
						`Profile này đang được mở! Vui lòng đóng trình duyệt hiện tại trước khi khởi chạy lại.`,
					);
				}
			}
		}

		if (!existsSync(join(this.profilePath, "orbita.config"))) {
			// Orbita browser tự động xoá orbita.config sau khi dùng, nên ta phải rebuild lại mỗi khi launch
			try {
				const prefsPath = join(this.profilePath, "Default", "Preferences");
				const zeroData = JSON.parse(readFileSync(prefsPath, "utf8"));
				const prefs = zeroData.gologin || {};

				const intl = {
					accept_languages: prefs.languages || "en-US",
					selected_languages: prefs.languages || "en-US",
					timezone: prefs.timezone?.id || "",
				};

				const clientOpts = {};
				for (const key of SECURED_ORBITA_OPTS) {
					if (key in prefs) clientOpts[key] = prefs[key];
				}

				let profileToken = ""; // Always fetch a fresh profile token

				if (!profileToken) {
					for (const auth of this.localAuths) {
						try {
							let pId = prefs.profile_id;
							try {
								if (pId)
									profileToken = await Api.fetchProfileToken(pId, auth.token);
							} catch (e: any) {
								console.error("Caught error:", e.message);
								// 403 or invalid old profile_id, fallback to fetch a new one
							}

							if (!profileToken) {
								pId = await Api.fetchFirstProfileId(auth.token);
								profileToken = await Api.fetchProfileToken(pId, auth.token);
								zeroData.gologin.profile_id = pId; // Update old Preferences with new active ID
							}

							if (profileToken) {
								// Save it back to Preferences so we don't fetch it again!
								zeroData.gologin.profile_token = profileToken;
								writeFileSync(prefsPath, JSON.stringify(zeroData));
								break;
							}
						} catch (e: any) {
							console.error("Caught error:", e.message);
							// Bỏ qua lỗi 403 để thử token tiếp theo
						}
					}
				}

				if (!profileToken) {
					throw new Error(
						"Không thể kết nối tới GoLogin API để lấy profile_token (tất cả token đều bị 403).",
					);
				}

				const orbitaConfig = {
					intl,
					gologin: Object.assign({ profile_token: profileToken }, clientOpts),
				};

				writeFileSync(
					join(this.profilePath, "orbita.config"),
					JSON.stringify(orbitaConfig, null, "\t"),
					"utf8",
				);
				console.log(
					`[INFO] Đã tự động tạo lại file orbita.config cho profile cũ.`,
				);
			} catch (err) {
				throw new Error(
					`File 'orbita.config' bị thiếu và không thể khôi phục tự động (${err.message}). Vui lòng tạo profile mới.`,
				);
			}
		}

		// Update port in Preferences so external scripts can connect to it
		const prefsPath = join(this.profilePath, "Default", "Preferences");
		const zeroData = JSON.parse(readFileSync(prefsPath, "utf8"));
		if (!zeroData.gologin) {
			zeroData.gologin = {};
		}
		zeroData.gologin.port = this.debugPort;

		writeFileSync(prefsPath, JSON.stringify(zeroData));

		const [w, h] = this.resolution.split("x");
		const args = [
			`--remote-debugging-port=${this.debugPort}`,
			`--password-store=basic`,
			`--gologin-profile=${this.profileName}`,
			`--lang=en-US`,
			`--webrtc-ip-handling-policy=default_public_interface_only`,
			`--disable-features=PrintCompositorLPAC`,
			`--user-data-dir=${this.profilePath}`,
			`--restore-last-session`,
		];

		if (!gridConfig) {
			args.push(`--window-size=${w},${h}`);
		}

		console.log(`[LAUNCH] Executable: ${BROWSER_EXE}`);
		console.log(`[LAUNCH] Arguments:\n  ${args.join("\n  ")}`);

		const proc = spawn(BROWSER_EXE, args, { detached: true, stdio: "ignore" });
		proc.unref();

		proc.on("exit", () => {
			const id = this.existingSessionId;
			console.log(`[PROFILE] Browser cho profile ${id} đã đóng.`);
			freeSlot(id);

			// Notify renderer: browser closed
			profileBus.emit("profile", { type: "profile:closed", id });

			// Dọn dẹp async — thông báo trạng thái backing-up / backed-up
			(async () => {
				try {
					// 1. Xoá port khỏi Preferences ngay lập tức
					const prefsPath = join(this.profilePath, "Default", "Preferences");
					if (existsSync(prefsPath)) {
						const prefs = JSON.parse(readFileSync(prefsPath, "utf8"));
						if (prefs.gologin?.port) {
							prefs.gologin.port = null;
							writeFileSync(prefsPath, JSON.stringify(prefs));
							console.log(`[PROFILE] Đã dọn dẹp port cho profile ${id}`);
						}
					}

					// 2. Báo đang dọn dẹp (cache, log, temp)
					profileBus.emit("profile", { type: "profile:backing-up", id });

					await sanitizeProfile(this.profilePath);

					console.log(`[PROFILE] Đã dọn dẹp xong profile ${id}`);
				} catch (e: any) {
					console.log(`[PROFILE] Lỗi khi dọn dẹp: ${e.message}`);
				} finally {
					// 3. Báo xong dù thành công hay lỗi
					profileBus.emit("profile", { type: "profile:backed-up", id });
				}
			})();
		});

		const success = await waitForDebugPort(this.debugPort, 7000);

		if (success) {
			console.log(
				`[OK] Profile mở thành công và Debug Port (${this.debugPort}) đã sẵn sàng!`,
			);
		} else {
			console.log(
				`[WARN] Không thể kết nối tới Debug Port sau 4s. Trình duyệt khởi động chậm hoặc đã bị đóng (VD: Lỗi JWT).`,
			);
		}
	}
}

export {
	Api,
	ConfigBuilder,
	GOLOGIN_DIR,
	getFreePort,
	OrbitaLauncher,
	waitForDebugPort,
};
