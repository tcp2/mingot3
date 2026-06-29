import { app, BrowserWindow, Tray, Menu, nativeImage } from "electron";
import path from "node:path";
import { registerIpcHandlers } from "./ipc";

// esbuild CJS output provides __dirname natively
const _dirname: string =
	typeof __dirname !== "undefined" ? __dirname : (process as any).cwd();

const isDev = process.env.ELECTRON_ENV === "dev";

let mainWindow: BrowserWindow | null = null;
let tray: Tray | null = null;

// In production, set APP_ROOT so Orbita stores profiles in userData
if (!isDev) {
	process.env.APP_ROOT = app.getPath("userData");
}

function getIconPath(): string {
	return path.join(_dirname, "public/icon.png");
}

function getPreloadPath(): string {
	// preload-main.cjs is compiled separately (see package.json electron:compile)
	return path.join(_dirname, "preload-main.cjs");
}

function createWindow(url: string) {
	mainWindow = new BrowserWindow({
		width: 1280,
		height: 800,
		minWidth: 960,
		minHeight: 600,
		webPreferences: {
			nodeIntegration: false,
			contextIsolation: true,
			// Preload script exposes window.electronAPI safely
			preload: getPreloadPath(),
		},
		title: "Mingot",
		icon: getIconPath(),
		show: false,
		backgroundColor: "#0f0f1a",
	});

	mainWindow.loadURL(url);
	mainWindow.setMenuBarVisibility(false);

	mainWindow.once("ready-to-show", () => {
		mainWindow?.show();
		if (isDev) {
			mainWindow?.webContents.openDevTools({ mode: "detach" });
		}
	});

	// Minimize to tray instead of closing
	mainWindow.on("close", (e) => {
		if (tray) {
			e.preventDefault();
			mainWindow?.hide();
		}
	});

	mainWindow.on("closed", () => {
		mainWindow = null;
	});
}

function createTray(url: string) {
	const icon = nativeImage.createFromPath(getIconPath());
	tray = new Tray(
		icon.isEmpty()
			? nativeImage.createEmpty()
			: icon.resize({ width: 16, height: 16 }),
	);

	const contextMenu = Menu.buildFromTemplate([
		{
			label: "Mở Mingot",
			click: () => {
				if (mainWindow) {
					mainWindow.show();
					mainWindow.focus();
				} else {
					createWindow(url);
				}
			},
		},
		{ type: "separator" },
		{
			label: "Thoát",
			click: () => {
				tray?.destroy();
				tray = null;
				app.quit();
			},
		},
	]);

	tray.setToolTip("Mingot - Profile Manager");
	tray.setContextMenu(contextMenu);
	tray.on("double-click", () => {
		mainWindow?.show();
		mainWindow?.focus();
	});
}

app.whenReady().then(async () => {
	// Register IPC handlers BEFORE creating the window
	// (works for both dev and production — in dev, preload is compiled but
	//  the renderer still uses fetch→Vite→Express for hot-reload convenience)
	registerIpcHandlers();

	let url: string;

	if (isDev) {
		// Dev: load Vite dev server (Express plugin handles /api/* via hot reload)
		url = "http://localhost:5173";
	} else {
		// Production: no HTTP server needed — frontend uses IPC exclusively
		url = `file://${path.join(_dirname, "dist/index.html")}`;
	}

	createWindow(url);
	createTray(url);

	app.on("activate", () => {
		if (BrowserWindow.getAllWindows().length === 0) {
			createWindow(url);
		} else {
			mainWindow?.show();
		}
	});
});

app.on("window-all-closed", () => {
	if (process.platform === "darwin") {
		app.quit();
	}
});
