<script lang="ts">
  import { toolCopy, toolGroups } from '../content/connected';
  import { setToolEnabled, type ConnectorSpace, type SpaceTool, type ToolGroup } from '../logic/connectors-api';

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
        {@const description = toolCopy[tool.name] ?? tool.description}
        <li class="flex flex-wrap items-center gap-3 py-2.5">
          <div class="min-w-44 flex-1 leading-tight">
            <p class="font-medium">{tool.title}</p>
            {#if description}<p class="text-xs text-[var(--brand-text-secondary)]">{description}</p>{/if}
          </div>
          <input
            class="toggle toggle-sm toggle-primary"
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
