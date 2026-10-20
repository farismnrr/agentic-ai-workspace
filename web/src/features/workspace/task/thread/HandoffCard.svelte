<script lang="ts">
  import DocText from '../../components/DocText.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import { effortLabel } from '../../logic/effort';
  import type { HandoffState } from '../../logic/task-types';

  interface Props {
    text: string;
    /** What the lead asked for, or what the teammate reported. */
    detail?: string | undefined;
    state: HandoffState;
    /** The teammate it was handed to, and the AI model they work with. */
    agent?: string | undefined;
    model?: string | undefined;
    /** How hard it was asked to think, shown beside the model. */
    effort?: string | undefined;
  }

  const { text, detail, state, agent, model, effort }: Props = $props();
  const engine = $derived(model === undefined ? '' : effort === undefined || effort === '' ? model : `${model} · ${effortLabel(effort).toLowerCase()}`);
  const icon = $derived(state === 'assigned' ? 'arrow-right' : state === 'reported' ? 'check-circle' : 'x-circle');
  const tone = $derived(state === 'reported' ? 'text-success' : state === 'failed' ? 'text-error' : 'text-[var(--brand-text-tertiary)]');
</script>

<details class="group rounded-lg border border-[var(--brand-border)] bg-base-100 px-3 py-2">
  <summary class="flex cursor-pointer list-none items-center gap-2">
    <Icon name={icon} class="size-4 shrink-0 {tone}" />
    {#if agent !== undefined}<span class="badge badge-outline badge-sm shrink-0">{agent}</span>{/if}
    <span class="min-w-0 flex-1 truncate font-medium">{text}</span>
    {#if engine !== '' && state === 'assigned'}<span class="hidden shrink-0 text-xs text-[var(--brand-text-tertiary)] sm:inline">{engine}</span>{/if}
    {#if detail !== undefined && detail !== ''}<Icon name="chevron-down" class="size-3.5 shrink-0 transition-transform group-open:rotate-180" />{/if}
  </summary>
  {#if detail !== undefined && detail !== ''}<div class="mt-2 rounded-md bg-base-200 px-3 py-2"><DocText text={detail} /></div>{/if}
</details>
