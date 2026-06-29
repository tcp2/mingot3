import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

export default {
	preprocess: vitePreprocess(),
	onwarn: (warning, handler) => {
		// Tắt toàn bộ warning của svelte để console sạch hơn
		return;
	},
};
