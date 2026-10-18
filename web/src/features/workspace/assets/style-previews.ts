// The preview pictures of the Designer's style families (drawn by scripts/style-previews.ts), found by family name.
// Vite hashes them, so a picture only changes its address when its content changes.
const found = import.meta.glob<string>('./style-previews/*.jpg', { eager: true, query: '?url', import: 'default' });

const pictures = new Map<string, string>(
  Object.entries(found).map(([path, url]) => [path.replace(/^.*\/(.*)\.jpg$/, '$1'), url]),
);

/** The picture of a style family, or undefined for a name without one. */
export const stylePreview = (name: string): string | undefined => pictures.get(name);
