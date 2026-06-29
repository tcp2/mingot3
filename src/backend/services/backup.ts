import { createWriteStream, existsSync, rmSync, readdirSync } from "node:fs";
import { join } from "node:path";
import yazl from "yazl";
import { GOLOGIN_DIR } from "./orbita";

export async function sanitizeProfile(profilePath: string): Promise<void> {
	const SEPARATOR = "/";

	// Directories to remove to save space
	const removeDirs = [
		`Default${SEPARATOR}Cache`,
		`Default${SEPARATOR}Code Cache`,
		`Default${SEPARATOR}GPUCache`,
		`Default${SEPARATOR}Service Worker`,
		`Default${SEPARATOR}IndexedDB`,
		`Default${SEPARATOR}DawnCache`,
		`Default${SEPARATOR}fonts_config`,
		`Default${SEPARATOR}Shared Dictionary${SEPARATOR}cache`,
		`GrShaderCache`,
		`ShaderCache`,
		`Crashpad`,
		`Guest Profile`,
		`System Profile`,
	];

	// Files to remove to save space / avoid locks
	const removeFiles = [
		`Default${SEPARATOR}lockfile`,
		`Default${SEPARATOR}History-journal`,
		`Default${SEPARATOR}Extension Cookies-journal`,
		`Default${SEPARATOR}Cookies-journal`,
		`Default${SEPARATOR}Web Data-journal`,
		`Default${SEPARATOR}QuotaManager-journal`,
		`Default${SEPARATOR}Network${SEPARATOR}Cookies-journal`,
		`Default${SEPARATOR}Network${SEPARATOR}Network Persistent State`,
		`Default${SEPARATOR}Network${SEPARATOR}Reporting and NEL`,
		`Default${SEPARATOR}Network${SEPARATOR}Reporting and NEL-journal`,
		`Default${SEPARATOR}Network${SEPARATOR}TransportSecurity`,
		`Default${SEPARATOR}Network${SEPARATOR}TrustTokens`,
		`Default${SEPARATOR}Network${SEPARATOR}TrustTokens-journal`,
		`chrome_debug.log`,
		`font_config_caches`,
		`CrashpadMetrics-active.pma`,
		`SingletonCookie`,
		`SingletonLock`,
		`SingletonSocket`,
	];

	for (const p of removeDirs) {
		const fullPath = join(profilePath, p);
		if (existsSync(fullPath)) {
			try {
				rmSync(fullPath, { recursive: true, force: true });
			} catch (e) {
				console.warn(`[BACKUP] Failed to remove dir: ${fullPath}`);
			}
		}
	}

	for (const p of removeFiles) {
		const fullPath = join(profilePath, p);
		if (existsSync(fullPath)) {
			try {
				rmSync(fullPath, { force: true });
			} catch (e) {
				console.warn(`[BACKUP] Failed to remove file: ${fullPath}`);
			}
		}
	}
}

export async function backupProfile(profileId: string): Promise<string | null> {
	try {
		const profilePath = join(GOLOGIN_DIR, profileId);
		const defaultFolder = join(profilePath, "Default");

		if (!existsSync(defaultFolder)) {
			console.warn(
				`[BACKUP] Profile ${profileId} does not have a Default folder.`,
			);
			return null;
		}

		// 1. Sanitize before backup
		await sanitizeProfile(profilePath);

		// 2. Setup backup directory
		const backupDir = GOLOGIN_DIR;

		// 3. Create zip
		return await new Promise((resolve) => {
			const zipPath = join(backupDir, `${profileId}.zip`);
			const zipfile = new yazl.ZipFile();

			zipfile.outputStream
				.pipe(createWriteStream(zipPath))
				.on("close", () => {
					console.log(
						`[BACKUP] Profile ${profileId} backed up successfully to profiles/${profileId}.zip`,
					);
					resolve(zipPath);
				})
				.on("error", (err: any) => {
					console.error(`[BACKUP] Archive error for ${profileId}:`, err);
					resolve(null);
				});

			// First Run marker as GoLogin does
			zipfile.addBuffer(Buffer.from(""), "First Run");

			// Zip the Default directory
			function getAllFiles(dir: string, baseDir = ""): string[] {
				let results: string[] = [];
				const list = readdirSync(dir, { withFileTypes: true });
				for (const dirent of list) {
					const relativePath = join(baseDir, dirent.name);
					const res = join(dir, dirent.name);
					if (dirent.isDirectory()) {
						results = results.concat(getAllFiles(res, relativePath));
					} else {
						results.push(relativePath);
					}
				}
				return results;
			}

			const files = getAllFiles(defaultFolder);
			for (const file of files) {
				const absPath = join(defaultFolder, file);
				const zipPathName = "Default/" + file.replace(/\\/g, "/");
				try {
					zipfile.addFile(absPath, zipPathName);
				} catch (e) {
					console.warn(`[BACKUP] Skipping non-file: ${absPath}`);
				}
			}

			zipfile.end();
		});
	} catch (error) {
		console.error(
			`[BACKUP] Unexpected error backing up profile ${profileId}:`,
			error,
		);
		return null;
	}
}

// Allow running directly via CLI for `exec` usage
if (process.argv[2] === "--run-backup" && process.argv[3]) {
	const profileId = process.argv[3];
	console.log(`[BACKUP-CLI] Starting backup for ${profileId}...`);
	backupProfile(profileId).then((success) => {
		process.exit(success ? 0 : 1);
	});
}
