<script lang="ts">
  import { currentTheme, toggleTheme, type Theme } from '../theme/theme';

  let theme = $state<Theme>(currentTheme());
  const isDark = $derived(theme === 'masihawam-dark');

  function handleToggle(): void {
    theme = toggleTheme();
  }
</script>

<button
  type="button"
  class="btn btn-ghost btn-sm gap-2 rounded-full border border-base-content/10 bg-base-100/70 backdrop-blur"
  aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
  aria-pressed={isDark}
  onclick={handleToggle}
>
  <span aria-hidden="true">{isDark ? '☀' : '◐'}</span>
  <span>{isDark ? 'Light' : 'Dark'}</span>
</button>
