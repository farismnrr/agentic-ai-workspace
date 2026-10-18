// Screenshots of the official Nuxt UI templates (their live demos), found by template name. Vite hashes them.
const found = import.meta.glob<string>('./template-previews/*.webp', { eager: true, query: '?url', import: 'default' });

const pictures = new Map<string, string>(
  Object.entries(found).map(([path, url]) => [path.replace(/^.*\/(.*)\.webp$/, '$1'), url]),
);

/** The screenshot of a template, or undefined for a name without one. */
export const templatePreview = (name: string): string | undefined => pictures.get(name);
