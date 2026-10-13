<script lang="ts">
  import { scorePassword } from '../validation';

  interface Props {
    password: string;
  }

  const { password }: Props = $props();
  const strength = $derived(scorePassword(password));
</script>

{#if password !== ''}
  <div class="flex items-center gap-3">
    <div class="flex flex-1 gap-1.5" aria-hidden="true">
      {#each [1, 2, 3, 4, 5] as step (step)}
        <span class="h-1.5 flex-1 rounded-full {step <= strength.score ? 'bg-primary' : 'bg-base-300'}"></span>
      {/each}
    </div>
    <p class="text-sm text-[var(--brand-text-secondary)]" role="status">{strength.label}</p>
  </div>
{/if}
