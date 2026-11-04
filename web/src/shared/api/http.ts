/** An error response from the backend; `code` is the stable machine-readable reason. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly title?: string | undefined;
  readonly detail?: string | undefined;

  constructor(status: number, code: string, title?: string, detail?: string) {
    super(detail ?? title ?? code);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.title = title;
    this.detail = detail;
  }
}

async function readBody(response: Response): Promise<unknown> {
  const text = await response.text();
  return text === '' ? undefined : JSON.parse(text);
}

function parseProblem(data: unknown): { code: string; title?: string | undefined; detail?: string | undefined } {
  if (typeof data === 'object' && data !== null) {
    const rec = data as Record<string, unknown>;
    const codeVal = rec['error'] ?? rec['code'];
    const code = typeof codeVal === 'string' ? codeVal : 'unknown';
    const titleVal = rec['title'];
    const title = typeof titleVal === 'string' ? titleVal : undefined;
    const detailVal = rec['detail'];
    const detail = typeof detailVal === 'string' ? detailVal : undefined;
    return { code, title, detail };
  }
  return { code: 'unknown' };
}

/** Sends a same-origin request, JSON or a form with files (cookies included) and returns the parsed body. */
export async function request<T>(method: string, path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  const init: RequestInit = {
    method,
    credentials: 'same-origin',
    ...(signal === undefined ? {} : { signal }),
  };
  if (body instanceof FormData) {
    init.body = body; // the browser adds the multipart Content-Type with its boundary
  } else if (body !== undefined) {
    init.headers = { 'Content-Type': 'application/json' };
    init.body = JSON.stringify(body);
  }
  const response = await fetch(path, init);
  const data = await readBody(response);
  if (!response.ok) {
    const problem = parseProblem(data);
    throw new ApiError(response.status, problem.code, problem.title, problem.detail);
  }
  return data as T;
}
