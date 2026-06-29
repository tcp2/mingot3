import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

// Set flag so server.mjs knows not to call app.listen()
process.env.VITE_DEV_SERVER = "true";

// https://vite.dev/config/
export default defineConfig({
	plugins: [svelte()],
	server: {
		watch: {
			ignored: ["**/profiles/**", "**/temp/**"],
		},
	},
});
