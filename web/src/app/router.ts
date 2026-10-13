import { writable } from 'svelte/store';

export type Route = 'landing' | 'login' | 'register';

const routes: Record<string, Route> = { '/': 'landing', '/login': 'login', '/register': 'register' };

const resolve = (path: string): Route => routes[path.replace(/(.)\/$/, '$1')] ?? 'landing';

export const route = writable<Route>(resolve(window.location.pathname));

export function navigate(path: string): void {
  window.history.pushState({}, '', path);
  route.set(resolve(path));
  window.scrollTo(0, 0);
}

export function syncFromLocation(): void {
  route.set(resolve(window.location.pathname));
}

const isPlainClick = (event: MouseEvent): boolean =>
  !event.defaultPrevented && event.button === 0 && !(event.metaKey || event.ctrlKey || event.shiftKey || event.altKey);

const internalAnchor = (event: MouseEvent): HTMLAnchorElement | undefined => {
  const anchor = event.target instanceof Element ? event.target.closest('a') : null;
  return anchor?.target === '' && anchor.origin === window.location.origin ? anchor : undefined;
};

/** Turns plain same-origin clicks on known routes into client-side navigation. */
export function interceptLinkClick(event: MouseEvent): void {
  const anchor = isPlainClick(event) ? internalAnchor(event) : undefined;
  if (anchor !== undefined && anchor.pathname in routes && anchor.hash === '') {
    event.preventDefault();
    navigate(anchor.pathname);
  }
}
