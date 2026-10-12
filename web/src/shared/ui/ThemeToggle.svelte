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
  class="btn btn-ghost btn-sm rounded-full border border-base-content/10"
  aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
  aria-pressed={isDark}
  onclick={handleToggle}
>
  {isDark ? 'Dark' : 'Light'}
</button>
