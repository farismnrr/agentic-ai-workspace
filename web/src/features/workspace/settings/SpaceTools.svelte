<script lang="ts">
  import { shortToolDescription, toolGroups } from '../content/connected';
  import { setToolEnabled, type ConnectorSpace, type SpaceTool, type ToolGroup } from '../logic/connectors-api';
  import Icon from '../../../shared/ui/Icon.svelte';

  interface Props {
    space: ConnectorSpace;
  }

  const { space }: Props = $props();
  const groups: ToolGroup[] = ['read', 'write', 'exec'];
  let overrides: Record<string, boolean> = $state({});
  let failed = $state(false);

  const key = (tool: SpaceTool): string => `${space.slug}:${tool.name}`;
  const isOn = (tool: SpaceTool): boolean => overrides[key(tool)] ?? tool.enabled;

  async function flip(tool: SpaceTool, enabled: boolean): Promise<void> {
    overrides[key(tool)] = enabled;
    failed = false;
    try {
      await setToolEnabled(space.slug, tool.name, enabled);
    } catch {
      overrides[key(tool)] = !enabled;
      failed = true;
    }
  }
</script>

{#if failed}<p class="border-b border-[var(--brand-border)] px-4 py-2 text-xs text-error" role="alert">We could not save that switch. Please try again.</p>{/if}
{#each groups as group (group)}
  {@const tools = space.tools.filter((tool) => tool.group === group)}
  {#if tools.length > 0}
    <div class="border-b border-[var(--brand-border)] bg-base-200/60 px-4 py-2">
      <h3 class="text-xs font-semibold">{toolGroups[group].title}</h3>
    </div>
    <ul class="divide-y divide-[var(--brand-border)] px-4">
      {#each tools as tool (tool.name)}
        {@const description = shortToolDescription(tool.name, tool.description)}
        <li class="flex items-center gap-3 py-2.5">
          <div class="group relative min-w-0 flex-1 leading-tight">
            <div class="flex min-w-0 items-center gap-1.5">
              <p class="truncate font-medium">{tool.title}</p>
              <button
                type="button"
                class="shrink-0 rounded-full text-[var(--brand-text-tertiary)] hover:text-[var(--brand-text-primary)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
                aria-label="Details about {tool.title}"
                aria-describedby="tool-detail-{space.slug}-{tool.name}"
              >
                <Icon name="info" class="size-3.5" />
              </button>
            </div>
            <p class="truncate text-xs text-[var(--brand-text-secondary)]">{description}</p>
            <span
              id="tool-detail-{space.slug}-{tool.name}"
              role="tooltip"
              class="pointer-events-none invisible absolute top-full left-0 z-50 mt-2 w-80 max-w-full rounded-lg border border-[var(--brand-border)] bg-base-100 p-3 text-left text-xs leading-relaxed font-normal whitespace-normal text-[var(--brand-text-primary)] opacity-0 shadow-lg transition-opacity group-hover:visible group-hover:opacity-100 group-focus-within:visible group-focus-within:opacity-100"
            >{tool.description.trim() || description}</span>
          </div>
          <input
            class="toggle toggle-sm toggle-primary shrink-0"
            type="checkbox"
            aria-label="Allow {tool.title}"
            checked={isOn(tool)}
            onchange={(event) => flip(tool, event.currentTarget.checked)}
          />
        </li>
      {/each}
    </ul>
  {/if}
{/each}
