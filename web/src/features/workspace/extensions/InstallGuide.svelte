<script lang="ts">
  import { extensionsCopy } from '../content/extensions-copy';
  import type { Extension } from '../logic/extensions-api';
  import Icon from '../../../shared/ui/Icon.svelte';

  interface Props {
    extension: Extension;
  }

  const { extension }: Props = $props();
  const copy = extensionsCopy.guide;

  type Browser = 'chrome' | 'edge';
  const addresses: Record<Browser, string> = { chrome: 'chrome://extensions', edge: 'edge://extensions' };
  const names: Record<Browser, string> = { chrome: 'Chrome', edge: 'Edge' };
  const guess = (): Browser => (typeof navigator !== 'undefined' && navigator.userAgent.includes('Edg/') ? 'edge' : 'chrome');

  let browser: Browser = $state(guess());
  let copied = $state(false);

  async function copyAddress(): Promise<void> {
    await navigator.clipboard.writeText(addresses[browser]);
    copied = true;
    setTimeout(() => { copied = false; }, 1500);
  }
</script>

<section class="rounded-xl border border-primary/30 bg-base-100 p-4" aria-label={copy.title(extension.name)} id="install-{extension.projectId}">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <h2 class="text-base font-semibold">{copy.title(extension.name)}</h2>
    <div class="join" role="group" aria-label={copy.browser}>
      {#each Object.keys(names) as key (key)}
        <button class="btn btn-xs join-item {browser === key ? 'btn-primary' : 'btn-outline'}" type="button" aria-pressed={browser === key} onclick={() => { browser = key as Browser; }}>{names[key as Browser]}</button>
      {/each}
    </div>
  </div>
  <ol class="mt-4 flex flex-col gap-4">
    <li class="flex gap-3">
      <span class="badge badge-primary badge-sm mt-1 shrink-0">1</span>
      <div class="min-w-0 flex-1">
        <p class="font-medium">{copy.steps.download.title}</p>
        <p class="text-sm text-[var(--brand-text-secondary)]">{copy.steps.download.body}</p>
        <a class="btn btn-primary btn-sm mt-2 rounded-lg" href={extension.downloadUrl} download={extension.fileName}><Icon name="download" class="size-4" /> {extension.fileName}</a>
      </div>
    </li>
    <li class="flex gap-3">
      <span class="badge badge-primary badge-sm mt-1 shrink-0">2</span>
      <div class="min-w-0 flex-1">
        <p class="font-medium">{copy.steps.open.title}</p>
        <p class="text-sm text-[var(--brand-text-secondary)]">{copy.steps.open.body}</p>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <code class="rounded-md bg-base-200 px-2 py-1 text-sm">{addresses[browser]}</code>
          <button class="btn btn-outline btn-xs" type="button" onclick={() => void copyAddress()}>{copied ? copy.copied : copy.copy}</button>
        </div>
      </div>
    </li>
    {#each [copy.steps.developer, copy.steps.load, copy.steps.pin] as step, index (step.title)}
      <li class="flex gap-3">
        <span class="badge badge-primary badge-sm mt-1 shrink-0">{index + 3}</span>
        <div class="min-w-0 flex-1">
          <p class="font-medium">{step.title}</p>
          <p class="text-sm text-[var(--brand-text-secondary)]">{step.body}</p>
        </div>
      </li>
    {/each}
  </ol>
  <p class="mt-4 flex gap-2 text-sm text-[var(--brand-text-secondary)]"><Icon name="info" class="mt-0.5 size-4 shrink-0" />{copy.warning}</p>
  <details class="mt-3 text-sm">
    <summary class="cursor-pointer font-medium">{copy.updateTitle}</summary>
    <p class="mt-1 text-[var(--brand-text-secondary)]">{copy.updateBody}</p>
  </details>
</section>
