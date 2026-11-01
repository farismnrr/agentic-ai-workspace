import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

// Keep svelte-check independent of Vite's bundled config loader:
// QA workspaces mount node_modules read-only, and Vite otherwise writes
// generated config modules into node_modules/.vite-temp.
export default {
  preprocess: vitePreprocess(),
};
