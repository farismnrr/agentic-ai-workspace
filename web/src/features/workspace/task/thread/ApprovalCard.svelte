<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import Notice from '../../components/Notice.svelte';
  import type { Risk } from '../../content/task-types';

  interface Props {
    text: string;
    action: string;
    risk: Risk;
    resolved?: 'allowed' | 'denied' | undefined;
  }

  const riskLabel: Record<Risk, string> = { low: 'Low risk', medium: 'Check first', high: 'High risk' };
  const riskTone: Record<Risk, string> = { low: 'badge-ghost', medium: 'badge-warning badge-soft', high: 'badge-error badge-soft' };
  const choices = ['Allow once', 'Always allow in this project', "Don't allow"];

  const { text, action, risk, resolved }: Props = $props();
  let picked = $state<string | undefined>(undefined);
</script>

<section class="card border border-warning/50 bg-base-100 p-3" aria-label="Permission request">
  <div class="flex items-start gap-2">
    <Icon name="shield" class="mt-0.5 size-5 shrink-0 text-warning" />
    <p class="flex-1 font-medium">{text}</p>
    <span class="badge badge-sm {riskTone[risk]}">{riskLabel[risk]}</span>
  </div>
  <p class="mt-3 rounded-lg bg-base-200 px-3 py-2 text-sm"><span class="text-[var(--brand-text-secondary)]">It will:</span> {action}</p>
  {#if resolved === undefined}
    <ul class="mt-3 flex flex-wrap gap-2">
      {#each choices as choice, index (choice)}
        <li><button class="btn btn-sm rounded-lg {index === 0 ? 'btn-primary' : 'btn-outline'}" type="button" onclick={() => (picked = choice)}>{choice}</button></li>
      {/each}
    </ul>
  {:else}
    <p class="mt-3 text-sm font-medium">You chose: {resolved === 'allowed' ? 'Allowed' : 'Not allowed'}</p>
  {/if}
  {#if picked !== undefined}<div class="mt-3"><Notice show /></div>{/if}
</section>
