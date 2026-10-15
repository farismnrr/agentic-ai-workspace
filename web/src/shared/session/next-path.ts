/** Reads the `next` query parameter, accepting only same-site paths. */
export function nextPath(search: string): string {
  const next = new URLSearchParams(search).get('next');
  return next !== null && next.startsWith('/') && !next.startsWith('//') ? next : '/app';
}
