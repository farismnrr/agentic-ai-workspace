import { ApiError, request } from '../../../shared/api/http';

export interface Attachment {
  id: string;
  file: File;
  path?: string;
  uploadError?: string;
}

interface Saved {
  path: string;
}

interface FileRule {
  kind: 'photo' | 'video' | 'document';
  limit: number;
  mime: string[];
}

export interface FileValidation {
  accepted: File[];
  errors: string[];
}

const megabyte = 1024 * 1024;
const maxFiles = 10;
const fileRules: Record<string, FileRule> = {
  png: { kind: 'photo', limit: 25 * megabyte, mime: ['image/png'] },
  jpg: { kind: 'photo', limit: 25 * megabyte, mime: ['image/jpeg'] },
  jpeg: { kind: 'photo', limit: 25 * megabyte, mime: ['image/jpeg'] },
  webp: { kind: 'photo', limit: 25 * megabyte, mime: ['image/webp'] },
  gif: { kind: 'photo', limit: 25 * megabyte, mime: ['image/gif'] },
  svg: { kind: 'photo', limit: 5 * megabyte, mime: ['image/svg+xml'] },
  pdf: { kind: 'document', limit: 25 * megabyte, mime: ['application/pdf'] },
  txt: { kind: 'document', limit: 25 * megabyte, mime: ['text/plain'] },
  md: { kind: 'document', limit: 25 * megabyte, mime: ['text/markdown', 'text/plain'] },
  csv: { kind: 'document', limit: 25 * megabyte, mime: ['text/csv', 'text/plain'] },
  json: { kind: 'document', limit: 25 * megabyte, mime: ['application/json', 'text/json', 'text/plain'] },
  log: { kind: 'document', limit: 25 * megabyte, mime: ['text/plain'] },
  mp4: { kind: 'video', limit: 100 * megabyte, mime: ['video/mp4'] },
  webm: { kind: 'video', limit: 100 * megabyte, mime: ['video/webm'] },
  mov: { kind: 'video', limit: 100 * megabyte, mime: ['video/quicktime'] },
};

function extensionOf(name: string): string {
  return name.split('.').at(-1)?.toLowerCase() ?? '';
}

export function attachmentKind(file: File): FileRule['kind'] | undefined {
  return fileRules[extensionOf(file.name)]?.kind;
}

export function validateFile(file: File): string | undefined {
  const rule = fileRules[extensionOf(file.name)];
  if (rule === undefined) return 'This file type is not supported.';
  if (file.type !== '' && file.type !== 'application/octet-stream' && !rule.mime.includes(file.type)) {
    return 'The file type does not match its name.';
  }
  if (file.size === 0) return 'This file is empty.';
  if (file.size > rule.limit) {
    const size = rule.kind === 'video' ? '100 MB' : rule.limit === 5 * megabyte ? '5 MB' : '25 MB';
    return `This file is over the ${size} limit. Choose a smaller file.`;
  }
  return undefined;
}

export function validateSelection(files: File[], currentCount: number): FileValidation {
  const accepted: File[] = [];
  const errors: string[] = [];
  for (const file of files) {
    if (currentCount + accepted.length >= maxFiles) {
      errors.push('You can attach up to 10 files to one message.');
      break;
    }
    const error = validateFile(file);
    if (error === undefined) accepted.push(file);
    else errors.push(`${file.name}: ${error}`);
  }
  return { accepted, errors };
}

export function mergeUploadResult(
  attachments: Attachment[],
  index: number,
  result: { path: string } | { error: string },
): Attachment[] {
  if (index < 0 || index >= attachments.length) return attachments;
  return attachments.map((attachment, at) => {
    if (at !== index) return attachment;
    return 'path' in result
      ? { id: attachment.id, file: attachment.file, path: result.path }
      : { id: attachment.id, file: attachment.file, uploadError: `${result.error} Press Send to try again.` };
  });
}

function uploadError(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 413 || error.code === 'too_large') return 'This file is too large.';
    if (error.status === 415 || ['unsupported_type', 'binary'].includes(error.code)) {
      return 'This file type is not supported.';
    }
    if (error.code === 'empty') return 'This file is empty.';
  }
  return 'Upload failed. Check your connection and try again.';
}

async function uploadOne(projectId: string, file: File): Promise<Saved> {
  const form = new FormData();
  form.append('file', file);
  return request<Saved>('POST', `/api/projects/${encodeURIComponent(projectId)}/uploads`, form);
}

/** Uploads pending files in order and returns paths/statuses for retry without repeating successes. */
export async function uploadAttachments(
  projectId: string,
  source: Attachment[],
  onProgress?: (completed: number, total: number) => void,
): Promise<Attachment[]> {
  let attachments: Attachment[] = source.map((item) => ({ ...item }));
  for (let index = 0; index < attachments.length; index += 1) {
    const item = attachments[index];
    if (item === undefined || item.path !== undefined) continue;
    try {
      const saved = await uploadOne(projectId, item.file);
      attachments = mergeUploadResult(attachments, index, saved);
    } catch (error) {
      attachments = mergeUploadResult(attachments, index, { error: uploadError(error) });
    }
    onProgress?.(index + 1, attachments.length);
  }
  return attachments;
}

/** Gives the task a useful prompt even when the person only attached files. */
export function withAttachments(text: string, paths: string[]): string {
  if (paths.length === 0) return text;
  const message = text.trim() === '' ? 'See the attached files.' : text;
  const files = paths.map((path) => `- ${path} — ${attachmentHint(path)}`).join('\n');
  return `${message}\n\nAttached files (already in the project):\n${files}`;
}

function attachmentHint(path: string): string {
  const extension = extensionOf(path);
  if (['png', 'jpg', 'jpeg', 'webp', 'gif', 'svg'].includes(extension)) {
    return 'view it with view_image (PNG only; convert other formats to PNG first)';
  }
  if (extension === 'pdf') return 'read it with pdftotext';
  if (['txt', 'md', 'csv', 'json', 'log'].includes(extension)) return 'read it directly';
  if (['mp4', 'webm', 'mov'].includes(extension)) {
    return 'stored only; you cannot watch video — tell the user what you can and cannot do';
  }
  return 'read it if you can';
}

export function formatFileSize(bytes: number): string {
  if (bytes < megabyte) return `${String(Math.max(1, Math.round(bytes / 1024)))} KB`;
  return `${(bytes / megabyte).toFixed(bytes >= 10 * megabyte ? 0 : 1)} MB`;
}

export function totalFileSize(attachments: Attachment[]): number {
  return attachments.reduce((total, item) => total + item.file.size, 0);
}
