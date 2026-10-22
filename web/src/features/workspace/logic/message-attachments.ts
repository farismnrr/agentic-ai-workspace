const marker = '\n\nAttached files (already in the project):\n';
const photoTypes = ['png', 'jpg', 'jpeg', 'webp', 'gif'];

export interface SentFile {
  path: string;
  name: string;
  photo: boolean;
}

/** Splits a sent message into what the person wrote and the files they attached (see `withAttachments`). */
export function splitAttachments(text: string): { text: string; files: SentFile[] } {
  const at = text.lastIndexOf(marker);
  if (at < 0) return { text, files: [] };
  const files = text
    .slice(at + marker.length)
    .split('\n')
    .flatMap((line) => {
      const path = /^- (.+?) — /.exec(line)?.[1];
      return path === undefined ? [] : [fileOf(path)];
    });
  if (files.length === 0) return { text, files: [] };
  const written = text.slice(0, at);
  return { text: written === 'See the attached files.' ? '' : written, files };
}

function fileOf(path: string): SentFile {
  const name = path.slice(path.lastIndexOf('/') + 1);
  const extension = name.slice(name.lastIndexOf('.') + 1).toLowerCase();
  return { path, name, photo: photoTypes.includes(extension) };
}
