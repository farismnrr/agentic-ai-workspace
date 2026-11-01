/** tus 1.0 browser transport for project ZIPs. One 8 MiB PATCH per request,
 * preserving the ordinary 11 MiB API body limit and server-verified offsets. */
const chunkBytes = 8 * 1024 * 1024;
const tus = '1.0.0';
export type ZIPProgress = (bytes: number, phase: 'uploading' | 'validating') => void;

interface ZIPStatus { offset: number; state: string; path?: string; }
interface UploadFailure extends Error { status?: number; }

function failure(response: Response): UploadFailure {
  const msg = response.status === 413 ? 'ZIP exceeds the upload limit.' :
    response.status === 507 ? 'Project storage quota is full.' :
    response.status === 409 ? 'Upload offset changed. Rechecking…' :
    response.status === 404 || response.status === 410 ? 'ZIP session expired or archive failed validation.' :
    response.status === 415 || response.status === 422 ? 'ZIP is invalid or unsafe.' :
    response.status === 401 || response.status === 403 ? 'You do not have access to upload this ZIP.' :
    `Upload failed (HTTP ${response.status}).`;
  const error = new Error(msg) as UploadFailure;
  error.status = response.status;
  return error;
}
async function send(method: string, url: string, headers: Record<string,string>,
  signal: AbortSignal, body?: BodyInit): Promise<Response> {
  const response = await fetch(url, { method, headers,
    ...(body === undefined ? {} : { body }), credentials:'same-origin', cache:'no-store', signal });
  if (!response.ok) throw failure(response);
  return response;
}

async function fingerprint(file: File): Promise<string> {
  const size = Math.min(65536, file.size);
  const sample = new Blob([file.slice(0, size), file.slice(Math.max(0, file.size - size))]);
  const hash = await crypto.subtle.digest('SHA-256', await sample.arrayBuffer());
  const hex = Array.from(new Uint8Array(hash), byte => byte.toString(16).padStart(2, '0')).join('');
  return `${file.name}:${file.size}:${file.lastModified}:${hex}`;
}
function cached(key: string): string | null {
  try { return sessionStorage.getItem(key); } catch { return null; }
}
function remember(key: string, value: string | null): void {
  try {
    if (value === null) sessionStorage.removeItem(key);
    else sessionStorage.setItem(key, value);
  } catch { /* sessionStorage may be disabled; current upload still works. */ }
}
function basename(projectId: string): string {
  return `/api/projects/${encodeURIComponent(projectId)}/uploads/resumable`;
}
async function create(base: string, file: File, signal: AbortSignal): Promise<string> {
  const encoded = new TextEncoder().encode(file.name);
  const name = btoa(Array.from(encoded, b => String.fromCharCode(b)).join(''));
  const response = await send('POST', base, {
    'Tus-Resumable': tus, 'Upload-Length': String(file.size), 'Upload-Metadata': `filename ${name}`,
  }, signal);
  const location = response.headers.get('Location');
  if (!location?.startsWith(base + '/')) throw new Error('Upload server returned an invalid location.');
  return location;
}
async function status(url: string, signal: AbortSignal): Promise<ZIPStatus> {
  const response = await send('HEAD', url, { 'Tus-Resumable': tus }, signal);
  const rawOffset = response.headers.get('Upload-Offset');
  if (rawOffset === null) throw new Error('Missing upload offset.');
  const offset = Number(rawOffset);
  if (!Number.isSafeInteger(offset) || offset < 0) throw new Error('Invalid upload offset.');
  const path = response.headers.get('Upload-Path');
  return { offset, state: response.headers.get('Upload-State') ?? 'uploading',
    ...(path === null ? {} : { path }) };
}
function recoverable(error: unknown): boolean {
  if (error instanceof DOMException && error.name === 'AbortError') return false;
  if (typeof error !== 'object' || error === null || !('status' in error)) return true;
  const code = (error as UploadFailure).status ?? 0;
  return code === 409 || code === 408 || code === 425 || code === 429 || (code >= 500 && code !== 507);
}
async function retryDelay(attempt: number, signal: AbortSignal): Promise<void> {
  const ms = Math.min(16000, 1000 * 2 ** attempt) * (0.75 + 0.5 * Math.random());
  await new Promise<void>((resolve, reject) => {
    if (signal.aborted) { reject(new DOMException('Paused', 'AbortError')); return; }
    const timer = setTimeout(() => { signal.removeEventListener('abort', aborted); resolve(); }, ms);
    function aborted(): void { clearTimeout(timer); reject(new DOMException('Paused', 'AbortError')); }
    signal.addEventListener('abort', aborted, { once:true });
  });
}

export async function uploadZIP(projectId: string, file: File, signal: AbortSignal,
  progress: ZIPProgress): Promise<string> {
  const base = basename(projectId);
  const key = `zip:v1:${projectId}:${await fingerprint(file)}`;
  let url = cached(key);
  let current: ZIPStatus | undefined;
  if (url !== null && !url.startsWith(base + '/')) { url = null; remember(key, null); }
  if (url !== null) {
    try { current = await status(url, signal); }
    catch (error) {
      if (signal.aborted) throw error;
      if (typeof error !== 'object' || error === null || !('status' in error) ||
          ![404,410].includes((error as UploadFailure).status ?? 0)) throw error;
      url = null; remember(key, null);
    }
  }
  if (url === null) { url = await create(base, file, signal); remember(key, url); }
  if (!current) current = await status(url, signal);
  if (current.offset > file.size) throw new Error('Server offset exceeds this file.');
  if (current.state === 'ready' && current.path) {
    remember(key, null); progress(file.size, 'uploading'); return current.path;
  }
  let attempts = 0;
  while (current.offset < file.size) {
    if (signal.aborted) throw new DOMException('Paused', 'AbortError');
    const end = Math.min(current.offset + chunkBytes, file.size);
    progress(current.offset, end === file.size ? 'validating' : 'uploading');
    try {
      const response = await send('PATCH', url, {
        'Tus-Resumable': tus, 'Upload-Offset': String(current.offset),
        'Content-Type': 'application/offset+octet-stream',
      }, signal, file.slice(current.offset, end));
      const acknowledged = Number(response.headers.get('Upload-Offset'));
      if (!Number.isSafeInteger(acknowledged) || acknowledged !== end) {
        throw new Error('Server returned an unexpected upload offset.');
      }
      current = { offset: acknowledged, state: response.headers.get('Upload-State') ?? 'uploading' };
      progress(current.offset, current.state === 'validating' ? 'validating' : 'uploading');
      attempts = 0;
    } catch (error) {
      if (signal.aborted || !recoverable(error) || attempts >= 5) throw error;
      await retryDelay(attempts++, signal);
      current = await status(url, signal);
    }
  }
  if (current.state !== 'ready' || !current.path) {
    for (let tries = 0; tries < 360 && (current.state !== 'ready' || !current.path); tries++) {
      progress(file.size, 'validating');
      await retryDelay(0, signal);
      current = await status(url, signal);
      if (!['validating', 'publishing', 'uploading', 'ready'].includes(current.state)) {
        throw new Error('ZIP validation failed.');
      }
    }
  }
  if (current.state !== 'ready' || !current.path) throw new Error('ZIP validation took too long. Retry to resume status polling.');
  remember(key, null);
  return current.path;
}

export async function discardZIP(projectId: string, file: File): Promise<void> {
  const key = `zip:v1:${projectId}:${await fingerprint(file)}`;
  const url = cached(key);
  if (url === null) return;
  remember(key, null);
  if (!url.startsWith(basename(projectId) + '/')) return;
  await fetch(url, { method: 'DELETE', credentials: 'same-origin',
    headers: { 'Tus-Resumable': tus }, cache: 'no-store' }).catch(() => undefined);
}
