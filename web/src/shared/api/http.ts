/** An error response from the backend; `code` is the stable machine-readable reason. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string) {
    super(code);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

async function readBody(response: Response): Promise<unknown> {
  const text = await response.text();
  return text === '' ? undefined : JSON.parse(text);
}

function errorCode(data: unknown): string {
  if (typeof data === 'object' && data !== null && 'error' in data && typeof data.error === 'string') return data.error;
  return 'unknown';
}

/** Sends a same-origin request, JSON or a form with files (cookies included) and returns the parsed body. */
export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method, credentials: 'same-origin' };
  if (body instanceof FormData) {
    init.body = body; // the browser adds the multipart Content-Type with its boundary
  } else if (body !== undefined) {
    init.headers = { 'Content-Type': 'application/json' };
    init.body = JSON.stringify(body);
  }
  const response = await fetch(path, init);
  const data = await readBody(response);
  if (!response.ok) throw new ApiError(response.status, errorCode(data));
  return data as T;
}
