/**
 * profileService.ts
 *
 * Pure business-logic layer — no Express req/res dependencies.
 * Called by both:
 *   - src/backend/routes/profiles.ts  (HTTP / standalone mode)
 *   - src/electron/ipc.ts             (Electron IPC mode)
 */

import {
	existsSync,
	globSync,
	mkdirSync,
	readFileSync,
	renameSync,
	cpSync,
	rmSync,
	statSync,
	writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import WebSocket from "ws";
import { GOLOGIN_DIR, OrbitaLauncher } from "./services/orbita";
import { backupProfile } from "./services/backup";
import { allocateSlot, freeSlot, getGridConfig } from "./services/gridTracker";
import yauzl from "yauzl";
import { dirname } from "node:path";
import { createWriteStream } from "node:fs";
import { monotonicFactory } from "ulid";

const ulid = monotonicFactory();

// ─── Helpers ──────────────────────────────────────────────────────────────────

export function isProfileRunning(profilePath: string): boolean {
	if (!existsSync(profilePath)) return false;
	try {
		renameSync(profilePath, profilePath);
		return false;
	} catch (e: any) {
		if (e.code === "EPERM" || e.code === "EBUSY") return true;
		return false;
	}
}

// ─── Types ────────────────────────────────────────────────────────────────────

export interface ProfileSummary {
	id: string;
	name: string;
	port: number | null;
	status: "running" | "stopped";
	folder: string;
}

export interface StartResult {
	port: number;
}

// ─── Service functions ────────────────────────────────────────────────────────

/** List all profiles in GOLOGIN_DIR */
export async function listProfiles(): Promise<ProfileSummary[]> {
	const dirs = (
		globSync("MG-*", { cwd: GOLOGIN_DIR } as any) as string[]
	).filter((entry) => {
		try {
			return statSync(join(GOLOGIN_DIR, entry)).isDirectory();
		} catch {
			return false;
		}
	});

	const profiles = await Promise.all(
		dirs.map(async (dir) => {
			const profilePath = join(GOLOGIN_DIR, dir);
			const prefsPath = join(profilePath, "Default", "Preferences");
			let name = dir;
			let port: number | null = null;
			let folder = "Uncategorized";

			if (existsSync(prefsPath)) {
				try {
					const prefs = JSON.parse(readFileSync(prefsPath, "utf8"));
					name = prefs.gologin?.name || dir;
					port = prefs.gologin?.port || null;
					folder = prefs.gologin?.folder || "Uncategorized";
				} catch (e) {
					console.warn(
						`[SERVICE] Could not parse Preferences for profile ${dir}`,
						e,
					);
				}
			}

			const running = isProfileRunning(profilePath);

			// Auto-cleanup: nếu backend còn giữ slot nhưng browser đã tắt → giải phóng
			if (!running) {
				freeSlot(dir);
			}

			const status: "running" | "stopped" = running ? "running" : "stopped";
			return { id: dir, name, port, status, folder };
		}),
	);

	profiles.sort((a, b) => b.id.localeCompare(a.id));
	return profiles;
}

/** Create a new blank profile */
export async function createProfile(): Promise<{ id: string; name: string }> {
	const launcher = new OrbitaLauncher();
	await launcher.init();
	launcher.buildProfile();
	return { id: launcher.localSessionId, name: launcher.profileName };
}

/** Start a profile, returns debug port */
export async function startProfile(
	id: string,
	screenWidth = 1440,
	screenHeight = 900,
): Promise<StartResult> {
	const slotIndex = allocateSlot(id);
	const gridConfig = getGridConfig(slotIndex, screenWidth, screenHeight);

	try {
		const launcher = new OrbitaLauncher(id);
		await launcher.init();
		launcher.buildProfile();
		await launcher.launch(gridConfig);
		return { port: launcher.debugPort };
	} catch (err) {
		freeSlot(id);
		throw err;
	}
}

/** Stop a profile via CDP Browser.close */
export async function stopProfile(id: string): Promise<void> {
	const profilePath = join(GOLOGIN_DIR, id);
	const prefsPath = join(profilePath, "Default", "Preferences");

	if (!existsSync(prefsPath)) {
		throw Object.assign(new Error("Port not found for this profile"), {
			code: "NOT_FOUND",
		});
	}

	const prefs = JSON.parse(readFileSync(prefsPath, "utf8"));
	const port = (prefs as any).gologin?.port;

	if (!port) {
		throw Object.assign(new Error("Port not found for this profile"), {
			code: "NOT_FOUND",
		});
	}

	// Graceful close via CDP
	try {
		const resVersion = await fetch(`http://127.0.0.1:${port}/json/version`);
		if (resVersion.ok) {
			const data = await resVersion.json();
			const wsUrl = data.webSocketDebuggerUrl;
			if (wsUrl) {
				await new Promise<void>((resolve) => {
					const ws = new WebSocket(wsUrl);
					ws.onopen = () => {
						ws.send(JSON.stringify({ id: 1, method: "Browser.close" }));
						setTimeout(resolve, 500);
					};
					ws.onerror = resolve as any;
				});
			}
		}
	} catch (cdpErr: any) {
		console.log("CDP stop warning:", cdpErr.message);
	}

	freeSlot(id);
}

/** Delete a profile directory */
export async function deleteProfile(id: string): Promise<void> {
	const profilePath = join(GOLOGIN_DIR, id);
	if (!existsSync(profilePath)) {
		throw Object.assign(new Error("Profile not found"), { code: "NOT_FOUND" });
	}
	if (isProfileRunning(profilePath)) {
		throw Object.assign(
			new Error("Cannot delete a running profile. Stop it first."),
			{ code: "RUNNING" },
		);
	}

	rmSync(profilePath, { recursive: true, force: true, maxRetries: 5 });

	const zipPath = join(GOLOGIN_DIR, `${id}.zip`);
	if (existsSync(zipPath)) rmSync(zipPath, { force: true, maxRetries: 5 });
}

/** Update folder for multiple profiles */
export async function updateFolder(
	ids: string[],
	folder: string,
): Promise<void> {
	for (const id of ids) {
		const prefsPath = join(GOLOGIN_DIR, id, "Default", "Preferences");
		if (existsSync(prefsPath)) {
			const prefs = JSON.parse(readFileSync(prefsPath, "utf8"));
			if (!prefs.gologin) prefs.gologin = {};
			prefs.gologin.folder = folder;
			writeFileSync(prefsPath, JSON.stringify(prefs));
		}
	}
}

/** Import a profile from a ZIP buffer */
export async function importProfile(
	zipData: Buffer | string,
	overwrite = false,
): Promise<{ id: string }> {
	const ts = Date.now();
	const tmpExtractDir = join(tmpdir(), `mg-import-${ts}`);
	let tmpZip = "";

	if (typeof zipData === "string") {
		tmpZip = zipData;
		console.log(`[SERVICE] importProfile started. source path:`, zipData);
	} else {
		tmpZip = join(tmpdir(), `mg-import-${ts}.zip`);
		console.log(
			`[SERVICE] importProfile started. zipBuffer length:`,
			zipData?.length,
		);
	}

	try {
		if (typeof zipData !== "string") {
			if (!Buffer.isBuffer(zipData) || !zipData.length) {
				console.error(`[SERVICE] importProfile failed: zipBuffer invalid`);
				throw new Error("No zip data received");
			}
			writeFileSync(tmpZip, zipData);
			console.log(`[SERVICE] importProfile wrote temp zip to`, tmpZip);
		}

		mkdirSync(tmpExtractDir, { recursive: true });

		console.log(`[SERVICE] importProfile extracting zip from path:`, tmpZip);
		await new Promise<void>((resolve, reject) => {
			yauzl.open(tmpZip, { lazyEntries: true }, (err, zipfile) => {
				console.log(
					`[SERVICE] yauzl.open callback fired. Err:`,
					err?.message,
					`Zipfile exists:`,
					!!zipfile,
				);
				if (err || !zipfile)
					return reject(err || new Error("Failed to open zip"));

				let entryCount = 0;
				zipfile.readEntry();

				zipfile.on("entry", (entry) => {
					entryCount++;
					// Too many logs could crash the terminal, just log every 100 entries or the first few
					if (entryCount <= 5 || entryCount % 100 === 0) {
						console.log(
							`[SERVICE] extracting entry #${entryCount}:`,
							entry.fileName,
						);
					}

					if (/\/$/.test(entry.fileName)) {
						mkdirSync(join(tmpExtractDir, entry.fileName), { recursive: true });
						zipfile.readEntry();
					} else {
						mkdirSync(join(tmpExtractDir, entry.fileName, ".."), {
							recursive: true,
						});
						zipfile.openReadStream(entry, (err, readStream) => {
							if (err || !readStream) return reject(err);
							const writeStream = createWriteStream(
								join(tmpExtractDir, entry.fileName),
							);
							writeStream.on("error", reject);
							writeStream.on("finish", () => {
								zipfile.readEntry();
							});
							readStream.pipe(writeStream);
						});
					}
					entryCount++;
				});
				zipfile.on("end", () => {
					console.log(
						`[SERVICE] yauzl finished extracting ${entryCount} entries.`,
					);
					resolve();
				});
				zipfile.on("error", (err) => {
					console.error(`[SERVICE] yauzl zipfile error:`, err);
					reject(err);
				});
			});
		});

		let sourceDir = tmpExtractDir;
		let finalId = `MG-${ulid()}`;

		// Attempt to read original ID from Preferences
		const prefsPath = join(tmpExtractDir, "Default", "Preferences");
		let originalId: string | null = null;
		if (existsSync(prefsPath)) {
			try {
				const prefs = JSON.parse(readFileSync(prefsPath, "utf8"));
				originalId =
					prefs.gologin?._id || prefs.gologin?.name || prefs._id || null;
			} catch (_) {}
		}

		if (originalId?.startsWith("MG-")) {
			finalId = originalId;
		}

		const extractedItems = globSync("*", {
			cwd: tmpExtractDir,
			onlyDirectories: true,
		} as any) as string[];
		if (extractedItems.length === 1 && extractedItems[0].startsWith("MG-")) {
			sourceDir = join(tmpExtractDir, extractedItems[0]);
			finalId = extractedItems[0];
		}

		const finalPath = join(GOLOGIN_DIR, finalId);
		console.log(`[SERVICE] importProfile final target ID will be:`, finalId);
		if (existsSync(finalPath)) {
			if (overwrite) {
				console.log(`[SERVICE] importProfile overwriting profile ${finalId}`);
				try {
					rmSync(finalPath, { recursive: true, force: true, maxRetries: 5 });
				} catch (e) {}
			} else {
				console.error(
					`[SERVICE] importProfile failed: profile ${finalId} already exists`,
				);
				try {
					rmSync(tmpExtractDir, {
						recursive: true,
						force: true,
						maxRetries: 5,
					});
				} catch (e) {
					console.warn("Could not delete tmpExtractDir", e);
				}
				throw Object.assign(
					new Error(
						`Profile "${finalId}" đã tồn tại. Xoá profile cũ trước rồi mới import lại.`,
					),
					{ code: "CONFLICT", profileId: finalId },
				);
			}
		}

		mkdirSync(GOLOGIN_DIR, { recursive: true });
		try {
			renameSync(tmpExtractDir, finalPath);
		} catch (err: any) {
			if (err.code === "EXDEV") {
				cpSync(tmpExtractDir, finalPath, { recursive: true });
				try {
					rmSync(tmpExtractDir, {
						recursive: true,
						force: true,
						maxRetries: 5,
					});
				} catch (e) {
					console.warn("Could not delete tmpExtractDir", e);
				}
			} else {
				throw err;
			}
		}
		console.log(`[SERVICE] importProfile moved files to final dir`);

		try {
			const finalPrefsPath = join(finalPath, "Default", "Preferences");
			if (existsSync(finalPrefsPath)) {
				const prefs = JSON.parse(readFileSync(finalPrefsPath, "utf8"));
				if (!prefs.gologin) prefs.gologin = {};
				if (!prefs.gologin.name) prefs.gologin.name = finalId;
				writeFileSync(finalPrefsPath, JSON.stringify(prefs));
			}
		} catch (_) {}

		console.log(`[SERVICE] Profile imported successfully as ${finalId}`);
		return { id: finalId };
	} catch (err) {
		console.error(`[SERVICE] importProfile error:`, err);
		throw err;
	} finally {
		if (typeof zipData !== "string" && tmpZip && existsSync(tmpZip)) {
			try {
				rmSync(tmpZip, { force: true });
			} catch (e) {}
		}
		if (existsSync(tmpExtractDir)) {
			try {
				rmSync(tmpExtractDir, { recursive: true, force: true });
			} catch (e) {}
		}
	}
}

/** Backup a profile to ZIP, returns the zip file path */
export async function exportProfileZip(id: string): Promise<string> {
	const profilePath = join(GOLOGIN_DIR, id);
	if (isProfileRunning(profilePath)) {
		throw new Error("Chỉ có thể backup khi profile đang tắt");
	}
	const zipPath = await backupProfile(id);
	if (!zipPath || !existsSync(zipPath)) {
		throw new Error("Backup failed");
	}
	return zipPath;
}

/** Arrange all running windows on screen */
export async function arrangeWindows(
	screenWidth = 1920,
	screenHeight = 1080,
): Promise<{ arrangedCount: number }> {
	const dirs = globSync("MG-*", {
		cwd: GOLOGIN_DIR,
		onlyDirectories: true,
	} as any) as any[] as string[];

	const runningProfiles: { id: string; port: number }[] = [];
	for (const id of dirs) {
		const profilePath = join(GOLOGIN_DIR, id);
		const prefsPath = join(profilePath, "Default", "Preferences");
		if (!existsSync(prefsPath)) continue;

		if (isProfileRunning(profilePath)) {
			try {
				const prefs = JSON.parse(readFileSync(prefsPath, "utf8"));
				const port = prefs.gologin?.port;
				if (port) runningProfiles.push({ id, port });
			} catch (_) {}
		}
	}

	runningProfiles.sort((a, b) => a.id.localeCompare(b.id));
	let arrangedCount = 0;

	for (let i = 0; i < runningProfiles.length; i++) {
		const { port } = runningProfiles[i];
		const gridConfig = getGridConfig(i, screenWidth, screenHeight);

		try {
			const resList = await fetch(`http://127.0.0.1:${port}/json/list`);
			if (!resList.ok) continue;
			const targets = await resList.json();
			const page = targets.find(
				(t: any) => t.type === "page" && t.webSocketDebuggerUrl,
			);
			if (!page) continue;

			await new Promise<void>((resolve) => {
				const ws = new WebSocket(page.webSocketDebuggerUrl);
				const timeout = setTimeout(() => {
					ws.close();
					resolve();
				}, 2000);

				ws.onopen = () => {
					ws.send(
						JSON.stringify({ id: 1, method: "Browser.getWindowForTarget" }),
					);
				};
				ws.onmessage = (event: any) => {
					const msg = JSON.parse(event.data);
					if (msg.id === 1) {
						if (msg.result?.windowId) {
							ws.send(
								JSON.stringify({
									id: 2,
									method: "Browser.setWindowBounds",
									params: {
										windowId: msg.result.windowId,
										bounds: {
											left: gridConfig.x,
											top: gridConfig.y,
											width: gridConfig.w,
											height: gridConfig.h,
											windowState: "normal",
										},
									},
								}),
							);
						} else {
							clearTimeout(timeout);
							ws.close();
							resolve();
						}
					} else if (msg.id === 2) {
						clearTimeout(timeout);
						ws.close();
						resolve();
					}
				};
				ws.onerror = () => {
					clearTimeout(timeout);
					resolve();
				};
			});
			arrangedCount++;
		} catch (err) {
			console.log(`Arrange failed for port ${port}:`, (err as any).message);
		}
	}

	return { arrangedCount };
}

// Allow running directly via CLI for testing
if (process.argv[2] === "--run-import" && process.argv[3]) {
	console.log(`[IMPORT-CLI] Starting import from ${process.argv[3]}...`);
	try {
		const buf = readFileSync(process.argv[3]);
		importProfile(buf)
			.then((result) => {
				console.log("[IMPORT-CLI] Success:", result);
				process.exit(0);
			})
			.catch((err) => {
				console.error("[IMPORT-CLI] Failed:", err);
				process.exit(1);
			});
	} catch (e) {
		console.error("[IMPORT-CLI] Failed to read file:", e);
		process.exit(1);
	}
}
