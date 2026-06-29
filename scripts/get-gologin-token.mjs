import { existsSync, readdirSync, readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

function findGologinStorage() {
	const candidates = [
		join(homedir(), ".gologin", "Local Storage", "leveldb"),
		resolve(
			homedir(),
			"AppData",
			"Roaming",
			"Gologin",
			"Local Storage",
			"leveldb",
		),
	];
	for (const dir of candidates) if (existsSync(dir)) return dir;
	return null;
}

export function getTokens() {
	const logDir = findGologinStorage();
	console.log("[DEBUG] Gologin Storage Dir:", logDir);

	if (!logDir) {
		console.log("[DEBUG] No storage directory found.");
		return [];
	}

	const files = readdirSync(logDir).filter(
		(f) => f.endsWith(".log") || f.endsWith(".ldb"),
	);
	console.log("[DEBUG] Found token files:", files);

	const results = [];

	for (const file of files) {
		const content = readFileSync(join(logDir, file), "binary");
		const matches = content.match(
			/eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+/g,
		);

		if (!matches) {
			console.log(`[DEBUG] File ${file}: 0 tokens found.`);
			continue;
		}

		const uniqueMatches = [...new Set(matches)];
		console.log(
			`[DEBUG] File ${file}: Found ${uniqueMatches.length} unique JWT matches.`,
		);

		for (const token of uniqueMatches) {
			try {
				const payload = JSON.parse(
					Buffer.from(token.split(".")[1], "base64url").toString(),
				);
				console.log(
					`[DEBUG] -> Parsed token successfully for User ID (sub): ${payload.sub}`,
				);
				results.push({ token, payload });
			} catch (err) {
				console.log(`[DEBUG] -> Failed to parse token:`, err.message);
			}
		}
	}

	console.log(`[DEBUG] Total valid tokens extracted: ${results.length}`);
	return results;
}

let isMain = false;
try {
	isMain =
		typeof import.meta !== "undefined" &&
		import.meta.url &&
		process.argv[1] === fileURLToPath(import.meta.url);
} catch (e) {}

if (isMain) {
	const logDir = findGologinStorage();
	if (!logDir) {
		console.error("Gologin Local Storage not found.");
		process.exit(1);
	}
	const tokens = getTokens();
	for (const { token, payload } of tokens) {
		console.log(JSON.stringify({ token, payload }, null, 2));
		console.log("---");
	}
}
