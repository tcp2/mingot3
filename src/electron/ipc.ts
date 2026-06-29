/**
 * ipc.ts
 *
 * Registers all ipcMain handlers for the Electron production mode.
 * Called once from main.ts after app is ready.
 *
 * Each handler mirrors an Express route but calls profileService directly.
 */

import { ipcMain, BrowserWindow } from "electron";
import { readFileSync, rmSync } from "node:fs";
import { profileBus } from "../backend/events";
import {
	listProfiles,
	createProfile,
	startProfile,
	stopProfile,
	deleteProfile,
	updateFolder,
	importProfile,
	exportProfileZip,
	arrangeWindows,
} from "../backend/profileService";

type IpcResult<T = void> =
	| { success: true; data?: T }
	| { success: false; error: string; code?: string; profileId?: string };

function ok<T>(data?: T): IpcResult<T> {
	return { success: true, data };
}

function fail(err: unknown, code?: string): IpcResult {
	const msg = err instanceof Error ? err.message : String(err);
	const profileId = (err as any)?.profileId;
	return { success: false, error: msg, code, profileId };
}

export function registerIpcHandlers(): void {
	// ── profiles:list ────────────────────────────────────────────────────────
	ipcMain.handle("profiles:list", async () => {
		try {
			console.log("[IPC] profiles:list requested");
			const profiles = await listProfiles();
			console.log("[IPC] profiles:list success:", profiles.length, "profiles");
			return ok(profiles);
		} catch (err) {
			console.error("[IPC] profiles:list error:", err);
			return fail(err);
		}
	});

	// ── profiles:create ──────────────────────────────────────────────────────
	ipcMain.handle("profiles:create", async () => {
		try {
			const profile = await createProfile();
			return ok(profile);
		} catch (err) {
			return fail(err);
		}
	});

	// ── profiles:start ───────────────────────────────────────────────────────
	ipcMain.handle(
		"profiles:start",
		async (_event, id: string, screenWidth?: number, screenHeight?: number) => {
			try {
				const result = await startProfile(id, screenWidth, screenHeight);
				return ok(result);
			} catch (err) {
				return fail(err);
			}
		},
	);

	// ── profiles:stop ────────────────────────────────────────────────────────
	ipcMain.handle("profiles:stop", async (_event, id: string) => {
		try {
			await stopProfile(id);
			return ok();
		} catch (err: any) {
			return fail(err, err?.code);
		}
	});

	// ── profiles:delete ──────────────────────────────────────────────────────
	ipcMain.handle("profiles:delete", async (_event, id: string) => {
		try {
			await deleteProfile(id);
			return ok();
		} catch (err: any) {
			return fail(err, err?.code);
		}
	});

	// ── profiles:folder ──────────────────────────────────────────────────────
	ipcMain.handle(
		"profiles:folder",
		async (_event, ids: string[], folder: string) => {
			try {
				await updateFolder(ids, folder);
				return ok();
			} catch (err) {
				return fail(err);
			}
		},
	);

	// ── profiles:import ──────────────────────────────────────────────────────
	// Receives Uint8Array (buffer) from the renderer
	ipcMain.handle(
		"profiles:import",
		async (_event, zipData: Uint8Array, overwrite = false) => {
			try {
				const buf = Buffer.from(zipData);
				const result = await importProfile(buf, overwrite);
				return ok(result);
			} catch (err: any) {
				return fail(err, err?.code);
			}
		},
	);

	// ── profiles:import_path ─────────────────────────────────────────────────
	// Receives a string path to a zip file from the renderer
	ipcMain.handle(
		"profiles:import_path",
		async (_event, path: string, overwrite = false) => {
			try {
				const result = await importProfile(path, overwrite);
				return ok(result);
			} catch (err: any) {
				return fail(err, err?.code);
			}
		},
	);

	// ── profiles:backup ──────────────────────────────────────────────────────
	// Returns the zip file as a Uint8Array so the renderer can trigger a download
	ipcMain.handle("profiles:backup", async (_event, id: string) => {
		try {
			const zipPath = await exportProfileZip(id);
			const buf = readFileSync(zipPath);
			try {
				rmSync(zipPath, { force: true });
			} catch (_) {}
			// Return as plain array so IPC can serialise it
			return ok(Array.from(buf));
		} catch (err) {
			return fail(err);
		}
	});

	// ── profiles:arrange ─────────────────────────────────────────────────────
	ipcMain.handle(
		"profiles:arrange",
		async (_event, screenWidth?: number, screenHeight?: number) => {
			try {
				const result = await arrangeWindows(screenWidth, screenHeight);
				return ok(result);
			} catch (err) {
				return fail(err);
			}
		},
	);

	// ── Forward backend events → renderer ────────────────────────────────────
	// When a browser closes itself (user clicks X), orbita.ts emits events
	// on profileBus. We forward them to all open renderer windows so the UI
	// updates immediately without waiting for the next poll cycle.
	profileBus.on("profile", (data) => {
		for (const win of BrowserWindow.getAllWindows()) {
			if (!win.isDestroyed()) {
				win.webContents.send("profile:event", data);
			}
		}
	});

	console.log("[IPC] All profile handlers registered.");
}
