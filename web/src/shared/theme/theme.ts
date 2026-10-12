export type Theme = 'masihawam-light' | 'masihawam-dark';

const LIGHT_THEME: Theme = 'masihawam-light';
const DARK_THEME: Theme = 'masihawam-dark';
const STORAGE_KEY = 'agentic-ai-workspace-theme';

function isTheme(value: string | null): value is Theme {
  return value === LIGHT_THEME || value === DARK_THEME;
}

function preferredTheme(): Theme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? DARK_THEME : LIGHT_THEME;
}

function applyTheme(theme: Theme): void {
  document.documentElement.dataset.theme = theme;
}

export function initTheme(): Theme {
  const savedTheme = window.localStorage.getItem(STORAGE_KEY);
  const theme = isTheme(savedTheme) ? savedTheme : preferredTheme();

  applyTheme(theme);

  return theme;
}

export function currentTheme(): Theme {
  const theme = document.documentElement.dataset.theme;

  return isTheme(theme) ? theme : preferredTheme();
}

export function toggleTheme(): Theme {
  const nextTheme = currentTheme() === DARK_THEME ? LIGHT_THEME : DARK_THEME;

  applyTheme(nextTheme);
  window.localStorage.setItem(STORAGE_KEY, nextTheme);

  return nextTheme;
}
