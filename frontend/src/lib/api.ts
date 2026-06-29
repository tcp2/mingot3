/**
 * api.ts — Frontend API abstraction layer
 *
 * Uses Wails IPC exclusively.
 *
 * Usage:
 *   import { api } from '../lib/api'
 *   const { profiles } = await api.listProfiles()
 */

/** Low-level Wails wrapper */
async function wailsCall(methodName: string, ...args: unknown[]): Promise<any> {
	if (typeof window === "undefined" || !(window as any).go?.main?.App) {
		throw new Error("Wails API is not available");
	}
	const result = await (window as any).go.main.App[methodName](...args);
	if (!result.success) {
		const err = Object.assign(new Error(result.error ?? "API error"), {
			code: result.code,
			profileId: result.profileId,
		});
		throw err;
	}
	return result.data;
}

// ─── Public API ───────────────────────────────────────────────────────────────

export type ProfileSummary = {
	id: string;
	name: string;
	port: number | null;
	status: "running" | "stopped";
	folder: string;
};

export const api = {
	/** List all profiles */
	async listProfiles(): Promise<{ profiles: ProfileSummary[] }> {
		const profiles = await wailsCall("ListProfiles");
		return { profiles };
	},

	/** Create a new blank profile */
	async createProfile(): Promise<{ profile: { id: string; name: string } }> {
		const profile = await wailsCall("CreateProfile");
		return { profile };
	},

	/** Start a profile browser */
	async startProfile(
		id: string,
		screenWidth = window.screen.availWidth,
		screenHeight = window.screen.availHeight,
	): Promise<{ port: number }> {
		return await wailsCall("StartProfile", id, screenWidth, screenHeight);
	},

	/** Stop a running profile */
	async stopProfile(id: string): Promise<void> {
		await wailsCall("StopProfile", id);
	},

	/** Delete a profile */
	async deleteProfile(id: string): Promise<void> {
		await wailsCall("DeleteProfile", id);
	},

	/** Move profiles to a folder */
	async updateFolder(ids: string[], folder: string): Promise<void> {
		await wailsCall("UpdateFolder", ids, folder);
	},

	/** Import a profile from a ZIP file (File object from <input type="file">) */
	async importProfile(file: File, overwrite = false): Promise<{ id: string }> {
		const ab = await file.arrayBuffer();
		let binary = '';
		const bytes = new Uint8Array(ab);
		const len = bytes.byteLength;
		for (let i = 0; i < len; i++) {
			binary += String.fromCharCode(bytes[i]);
		}
		const base64 = window.btoa(binary);
		return await wailsCall("ImportProfile", base64, overwrite);
	},

	/** Export (backup) a profile as ZIP — triggers a browser download */
	async backupProfile(id: string): Promise<void> {
		const b64 = await wailsCall("ExportProfile", id);
		const binaryString = window.atob(b64);
		const bytes = new Uint8Array(binaryString.length);
		for (let i = 0; i < binaryString.length; i++) {
			bytes[i] = binaryString.charCodeAt(i);
		}
		const blob = new Blob([bytes], { type: "application/zip" });
		triggerDownload(blob, `${id}.zip`);
	},

	/** Arrange all running browser windows on screen */
	async arrangeWindows(
		screenWidth = window.screen.availWidth,
		screenHeight = window.screen.availHeight,
	): Promise<void> {
		await wailsCall("ArrangeWindows", screenWidth, screenHeight);
	},
};

// ─── Helpers ──────────────────────────────────────────────────────────────────

function triggerDownload(blob: Blob, filename: string) {
	const url = URL.createObjectURL(blob);
	const a = document.createElement("a");
	a.href = url;
	a.download = filename;
	document.body.appendChild(a);
	a.click();
	URL.revokeObjectURL(url);
	a.remove();
}

// ─── Push events (Wails) ─────────────────────────────────────────────────────

export type ProfileEventType =
	| "profile:closed"
	| "profile:backing-up"
	| "profile:backed-up";
export type ProfileEvent = { type: ProfileEventType; id: string };

/**
 * Subscribe to real-time profile events pushed from the main process.
 * Returns an unsubscribe function.
 */
export function onProfileEvent(
	listener: (event: ProfileEvent) => void,
): () => void {
	if (typeof window !== "undefined" && (window as any).runtime?.EventsOn) {
		const handler = (data: ProfileEvent) => listener(data);
		(window as any).runtime.EventsOn("profile:event", handler);
		return () => (window as any).runtime.EventsOff("profile:event");
	}
	return () => {};
}
