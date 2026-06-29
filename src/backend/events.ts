/**
 * src/backend/events.ts
 *
 * Lightweight event bus — decouples backend services (orbita, profileService)
 * from Electron IPC. Backend emits events here; ipc.ts forwards them to the renderer.
 */

import { EventEmitter } from "node:events";

export type ProfileEvent =
	| { type: "profile:closed"; id: string }
	| { type: "profile:backing-up"; id: string }
	| { type: "profile:backed-up"; id: string };

class ProfileEventBus extends EventEmitter {
	emit(event: "profile", data: ProfileEvent): boolean {
		return super.emit(event, data);
	}
	on(event: "profile", listener: (data: ProfileEvent) => void): this {
		return super.on(event, listener);
	}
	off(event: "profile", listener: (data: ProfileEvent) => void): this {
		return super.off(event, listener);
	}
}

export const profileBus = new ProfileEventBus();
