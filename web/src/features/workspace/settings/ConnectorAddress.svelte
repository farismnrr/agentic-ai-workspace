<script lang="ts">
  interface Props {
    url: string;
  }

  const { url }: Props = $props();
  let copied = $state(false);

  async function copy(): Promise<void> {
    try {
      await navigator.clipboard.writeText(url);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    } catch {
      copied = false;
    }
  }
</script>

<div class="border-b border-[var(--brand-border)] px-4 py-3">
  <label class="text-xs font-medium" for="connector-address">Connector address</label>
  <div class="mt-1 flex gap-2">
    <input id="connector-address" class="input input-sm w-full font-mono text-xs" readonly value={url} onfocus={(event) => { event.currentTarget.select(); }} />
    <button class="btn btn-sm" type="button" onclick={copy}>{copied ? 'Copied' : 'Copy'}</button>
  </div>
</div>
