<script lang="ts">
  import { onMount } from 'svelte';

  import { createAgentGraph, type AgentGraphController } from '../visual/agentGraph';

  const labels = [
    { className: 'left-[7%] top-[30%]', eyebrow: 'repo', label: 'src / app' },
    { className: 'left-[12%] bottom-[25%]', eyebrow: 'context', label: 'AGENTS.md' },
    { className: 'right-[10%] top-[27%]', eyebrow: 'tool', label: 'terminal' },
    { className: 'right-[6%] bottom-[28%]', eyebrow: 'proof', label: 'tests passing' },
  ] as const;

  let canvas: HTMLCanvasElement;
  let host: HTMLDivElement;
  let controller: AgentGraphController | null = null;
  let supported = $state(true);

  function handlePointer(event: PointerEvent): void {
    if (controller === null) return;
    const bounds = host.getBoundingClientRect();
    const x = ((event.clientX - bounds.left) / bounds.width) * 2 - 1;
    const y = ((event.clientY - bounds.top) / bounds.height) * 2 - 1;
    controller.pointer(x, y);
  }

  function clearPointer(): void {
    controller?.pointer(0, 0);
  }

  onMount(() => {
    controller = createAgentGraph(canvas);
    supported = controller !== null;
    if (controller === null) return;

    const resizeObserver = new ResizeObserver(([entry]) => {
      if (entry === undefined) return;
      controller?.resize(entry.contentRect.width, entry.contentRect.height);
    });
    const themeObserver = new MutationObserver(() => controller?.refreshTheme());

    resizeObserver.observe(host);
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
    controller.resize(host.clientWidth, host.clientHeight);

    return () => {
      resizeObserver.disconnect();
      themeObserver.disconnect();
      controller?.destroy();
      controller = null;
    };
  });
</script>

<div
  bind:this={host}
  class="graph-host absolute inset-0"
  onpointermove={handlePointer}
  onpointerleave={clearPointer}
  role="presentation"
>
  <canvas bind:this={canvas} class="absolute inset-0 size-full" aria-hidden="true"></canvas>

  {#if !supported}
    <div class="absolute inset-[16%] rounded-full border border-primary/30 opacity-60" aria-hidden="true"></div>
    <div class="absolute inset-[28%] rotate-45 border border-secondary/30" aria-hidden="true"></div>
  {/if}

  <div class="pointer-events-none absolute inset-0 hidden md:block" aria-hidden="true">
    {#each labels as item (item.label)}
      <div class={`graph-label absolute ${item.className}`}>
        <span class="block font-mono text-[0.58rem] uppercase tracking-[0.2em] text-[var(--brand-text-tertiary)]">{item.eyebrow}</span>
        <span class="mt-1 block text-xs font-semibold">{item.label}</span>
      </div>
    {/each}
  </div>

  <div class="pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 text-center" aria-hidden="true">
    <span class="font-mono text-[0.6rem] font-semibold uppercase tracking-[0.28em] text-primary">agent core</span>
  </div>
</div>

<style>
  .graph-host {
    mask-image: linear-gradient(to bottom, transparent 0%, black 10%, black 82%, transparent 100%);
  }

  .graph-label {
    min-width: 7rem;
    padding: 0.6rem 0.7rem;
    border-left: 1px solid var(--brand-border-strong);
    background: linear-gradient(90deg, var(--brand-surface-soft), transparent);
    backdrop-filter: blur(10px);
  }
</style>
