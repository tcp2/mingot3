#!/usr/bin/env node
/**
 * mingot.ts
 *
 * 1) Fetch fingerprint and config via GoLogin API (with fallback prioritizing healthy tokens)
 * 2) Fetch timezone from geo.myip.link
 * 3) Build & write config files: Preferences, orbita.config, etc.
 * 4) Manage Orbita browser lifecycle
 *
 * Usage:
 *   node mingot.ts create              # Only create a new profile, DO NOT launch browser
 *   node mingot.ts start               # Create a new profile AND launch browser
 *   node mingot.ts start MG-123...     # Load existing profile AND launch browser
 *   node mingot.ts MG-123...           # Shorthand for 'start' command with existing profile
 *
 * Options:
 *   --force                             # Force delete old config directory (if exists) when running
 *   -c, --createOnly                    # Only create a new profile, DO NOT launch browser
 */
import { defineCommand, runMain } from "citty";
import { OrbitaLauncher } from "./services/orbita";
import { backupProfile } from "./services/backup";

// ─── MAIN ─────────────────────────────────────────────────────────────────────
const main = defineCommand({
	meta: {
		name: "mingot",
		description: "Manage Orbita browser lifecycle",
	},
	args: {
		force: {
			type: "boolean",
			description: "Force delete old config directory (if exists) when running",
		},
		createOnly: {
			type: "boolean",
			alias: "c",
			description: "Only create a new profile, DO NOT launch browser",
		},
		cmdOrSession: {
			type: "positional",
			description:
				"Command (start | create | backup) or Session ID (e.g., MG-123...)",
			required: false,
		},
		sessionId: {
			type: "positional",
			description: "Session ID (if command is start)",
			required: false,
		},
	},
	async run({ args }) {
		try {
			const arg0 = args.cmdOrSession;
			const arg1 = args.sessionId;

			const command =
				args.createOnly || arg0 === "create"
					? "create"
					: arg0 === "backup"
						? "backup"
						: "start";

			let sessionId = null;
			if (arg0 === "start" || arg0 === "backup") {
				sessionId = arg1;
			} else if (arg0 && arg0 !== "create") {
				sessionId = arg0;
			}

			if (
				sessionId &&
				!sessionId.startsWith("MG-") &&
				!sessionId.startsWith("P")
			) {
				throw new Error(`Invalid command or session ID format: ${arg0}`);
			}

			// Initialize launcher based on parsed logic
			let launcher: OrbitaLauncher | undefined;
			if (command !== "backup") {
				if (sessionId) {
					launcher = new OrbitaLauncher(sessionId);
				} else {
					launcher = new OrbitaLauncher();
				}

				await launcher.init();
				launcher.buildProfile(args.force);
			}

			// Execute final step based on the command
			if (command === "backup") {
				if (!sessionId) {
					throw new Error("Session ID is required for backup");
				}
				console.log(`\n[INFO] Starting backup for profile ${sessionId}...`);
				const success = await backupProfile(sessionId);
				process.exit(success ? 0 : 1);
			} else if (command === "create") {
				console.log(
					"\n[INFO] Profile config created/updated. Skipping browser launch.",
				);
			} else if (launcher) {
				await launcher.launch();
			}
		} catch (err) {
			console.error("\n[FATAL ERROR]", err.message);
			process.exit(1);
		}
	},
});

runMain(main);
