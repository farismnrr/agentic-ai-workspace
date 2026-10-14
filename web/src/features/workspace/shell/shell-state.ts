import { writable } from 'svelte/store';

/** Desktop sidebar is shrunk to an icon rail. */
export const sidebarCollapsed = writable(false);

/** Command palette (Ctrl/Cmd+K) is open. */
export const paletteOpen = writable(false);

export interface Crumb {
  label: string;
  href?: string;
}
