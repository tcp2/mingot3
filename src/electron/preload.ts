/**
 * preload.ts
 *
 * Runs in renderer context with Node.js access, but isolated from page JS.
 * Exposes a safe `window.electronAPI` bridge via contextBridge.
 */

import { contextBridge, ipcRenderer } from "electron";

const electronAPI = {
	/** Invoke an IPC handler in the main process */
	invoke: (channel: string, ...args: unknown[]): Promise<unknown> => {
		return ipcRenderer.invoke(channel, ...args);
	},

	/** Subscribe to push events from the main process */
	on: (channel: string, listener: (...args: unknown[]) => void): void => {
		ipcRenderer.on(channel, (_event, ...args) => listener(...args));
	},

	/** Unsubscribe from push events */
	off: (channel: string, _listener: (...args: unknown[]) => void): void => {
		ipcRenderer.removeAllListeners(channel);
	},
};

contextBridge.exposeInMainWorld("electronAPI", electronAPI);
