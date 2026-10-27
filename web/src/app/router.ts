import { writable } from 'svelte/store';
import { matchRoute, type Route } from './route-matcher';
export { matchRoute, type Route, type SettingsTab } from './route-matcher';

const pathOf = (url: string): string => url.split(/[?#]/)[0] ?? url;

const resolve = (url: string): Route => matchRoute(pathOf(url)) ?? { name: 'not-found' };

export const route = writable<Route>(resolve(window.location.pathname));

/**
 * Goes to a same-site path after sign-in. App routes stay inside the page; anything else, such as an
 * app under /run, is a real page load because the server (not the SPA) answers it. `replace` leaves no
 * history entry behind, so Back does not return to the page that was left.
 */
export function goTo(path: string, replace = false): void {
  if (matchRoute(pathOf(path)) !== undefined) navigate(path, replace);
  else if (replace) window.location.replace(path);
  else window.location.assign(path);
}

/** Moves to a same-site URL (query allowed); `replace` skips adding a history entry. */
export function navigate(path: string, replace = false): void {
  if (replace) window.history.replaceState({}, '', path);
  else window.history.pushState({}, '', path);
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
  if (anchor?.hash === '' && matchRoute(anchor.pathname) !== undefined) {
    event.preventDefault();
    navigate(anchor.pathname);
  }
}
