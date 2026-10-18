import { request } from '../../../shared/api/http';

interface Saved {
  path: string;
}

/** Saves one picture into the project's assets/ folder; fails with code "binary", "too_large" or "exists". */
export function uploadAsset(projectId: string, file: File): Promise<Saved> {
  const form = new FormData();
  form.append('file', file);
  return request<Saved>('POST', `/api/projects/${encodeURIComponent(projectId)}/assets`, form);
}

/** Uploads the pictures one after the other and returns where each was saved. */
export async function uploadAssets(projectId: string, files: File[]): Promise<string[]> {
  const saved: string[] = [];
  for (const file of files) saved.push((await uploadAsset(projectId, file)).path);
  return saved;
}

/** Tells the agents which files the person attached, so they use them instead of searching for pictures. */
export function withAttachments(text: string, paths: string[]): string {
  if (paths.length === 0) return text;
  return `${text}\n\nPictures I attached (already in the project): ${paths.join(', ')}`;
}
