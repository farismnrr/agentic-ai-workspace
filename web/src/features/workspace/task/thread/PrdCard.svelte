<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import DocText from '../../components/DocText.svelte';

  interface Props {
    text: string;
    /** The document, in markdown. */
    document: string;
    agent?: string | undefined;
  }

  const { text, document, agent }: Props = $props();
</script>

<details class="group rounded-lg border border-[var(--brand-border)] bg-base-100 px-3 py-2" open>
  <summary class="flex cursor-pointer list-none items-center gap-2">
    <Icon name="file-text" class="size-4 shrink-0 text-secondary" />
    {#if agent !== undefined}<span class="badge badge-outline badge-sm shrink-0">{agent}</span>{/if}
    <span class="min-w-0 flex-1 truncate font-medium">{text}</span>
    <Icon name="chevron-down" class="size-3.5 shrink-0 transition-transform group-open:rotate-180" />
  </summary>
  <div class="mt-2 max-h-[28rem] overflow-y-auto rounded-md bg-base-200 px-3 py-2"><DocText text={document} /></div>
</details>
