<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import type { Risk } from '../../logic/task-types';

  interface Props {
    text: string;
    action: string;
    risk: Risk;
    resolved?: 'allowed' | 'denied' | undefined;
    onanswer: (allow: boolean) => void;
  }

  const { text, action, risk, resolved, onanswer }: Props = $props();
</script>

<section class="card border border-warning/50 bg-base-100 p-3" aria-label="Permission request">
  <div class="flex items-start gap-2">
    <Icon name="shield" class="mt-0.5 size-5 shrink-0 text-warning" />
    <p class="flex-1 font-medium">{text}</p>
    {#if risk === 'high'}<span class="badge badge-sm badge-error badge-soft">Be careful</span>{/if}
  </div>
  <details class="mt-2 text-sm">
    <summary class="cursor-pointer text-[var(--brand-text-secondary)]">Show details</summary>
    <p class="mt-1 rounded-lg bg-base-200 px-3 py-2"><code class="break-all">{action}</code></p>
  </details>
  {#if resolved === undefined}
    <ul class="mt-3 flex flex-wrap gap-2">
      <li><button class="btn btn-primary btn-sm rounded-lg" type="button" onclick={() => { onanswer(true); }}>Allow</button></li>
      <li><button class="btn btn-outline btn-sm rounded-lg" type="button" onclick={() => { onanswer(false); }}>Don't allow</button></li>
    </ul>
  {:else}
    <p class="mt-3 text-sm font-medium">You chose: {resolved === 'allowed' ? 'Allowed' : 'Not allowed'}</p>
  {/if}
</section>
