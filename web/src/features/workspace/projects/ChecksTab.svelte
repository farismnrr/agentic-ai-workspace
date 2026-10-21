<script lang="ts">
  import EmptyState from '../../../shared/ui/EmptyState.svelte';
  import { onMount } from 'svelte';
  import Icon from '../../../shared/ui/Icon.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import { projectsCopy } from '../content/projects-copy';
  import { loadChecks, saveChecks, type Check } from '../logic/checks-api';
  import { joinCommand } from '../logic/command';
  import { describeProjectError } from '../logic/project-errors';
  import CheckModal from './CheckModal.svelte';

  interface Props {
    projectId: string;
  }

  const { projectId }: Props = $props();
  let checks: Check[] = $state([]);
  let loading = $state(true);
  let failed = $state(false);
  let editing: Check | undefined = $state();
  let formOpen = $state(false);
  let removing: Check | undefined = $state();
  let error = $state('');
  let busy = $state(false);

  async function load(): Promise<void> {
    try {
      checks = await loadChecks(projectId);
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  }

  async function persist(next: Check[]): Promise<boolean> {
    busy = true;
    error = '';
    try {
      await saveChecks(projectId, next);
      checks = next;
      return true;
    } catch (failure) {
      error = describeProjectError(failure);
      return false;
    } finally {
      busy = false;
    }
  }

  async function save(check: Check): Promise<void> {
    const others = checks.filter((item) => item.name !== editing?.name);
    if (await persist([...others, check].sort((a, b) => a.name.localeCompare(b.name)))) formOpen = false;
  }

  async function confirmRemove(): Promise<void> {
    const target = removing;
    removing = undefined;
    if (target !== undefined) await persist(checks.filter((item) => item.name !== target.name));
  }

  function open(check: Check | undefined): void {
    editing = check;
    error = '';
    formOpen = true;
  }

  onMount(load);
</script>

<p class="max-w-2xl text-[var(--brand-text-secondary)]">{projectsCopy.checks.intro}</p>
<p class="mt-1 max-w-2xl text-xs text-[var(--brand-text-tertiary)]">{projectsCopy.checks.warning}</p>

{#if loading}
  <span class="loading loading-spinner loading-sm mt-4" aria-label="Loading"></span>
{:else if failed}
  <p class="mt-4 text-error" role="alert">We could not load the checks. Please try again in a moment.</p>
{:else}
  <div class="mt-4">
    <ul class="divide-y divide-[var(--brand-border)]">
      {#each checks as check (check.name)}
        <li class="flex flex-wrap items-center gap-3 px-4 py-3">
          <div class="min-w-0 flex-1">
            <p class="font-medium"><code>{check.name}</code>{#if check.description !== ''} <span class="font-normal text-[var(--brand-text-secondary)]">· {check.description}</span>{/if}</p>
            <code class="mt-1 block truncate text-xs text-[var(--brand-text-secondary)]">{joinCommand(check.command)}{check.timeoutSeconds > 0 ? ` · up to ${String(check.timeoutSeconds)} s` : ''}</code>
          </div>
          <button class="btn btn-ghost btn-xs" type="button" onclick={() => { open(check); }}>Edit</button>
          <button class="btn btn-ghost btn-xs btn-square" type="button" aria-label="Remove {check.name}" onclick={() => (removing = check)}><Icon name="trash" class="size-4" /></button>
        </li>
      {:else}
        <li><EmptyState compact icon="check-square" title={projectsCopy.checks.empty.title} body="{projectsCopy.checks.empty.body} For example: {projectsCopy.checks.examples.join(', ')}." /></li>
      {/each}
    </ul>
    <div class="border-t border-[var(--brand-border)] py-3">
      <button class="btn btn-ghost btn-sm" type="button" onclick={() => { open(undefined); }}><Icon name="plus" class="size-4" /> Add a check</button>
    </div>
  </div>
  {#if error !== '' && !formOpen}<p class="mt-3 text-sm text-error" role="alert">{error}</p>{/if}
{/if}

<CheckModal check={editing} bind:show={formOpen} {error} {busy} onsave={save} />
<ConfirmModal
  open={removing !== undefined}
  title="Remove the {removing?.name ?? ''} check?"
  body="The AI will no longer be able to run it, so work it does cannot be verified with it."
  confirmLabel="Remove check"
  cancelLabel="Keep it"
  oncancel={() => (removing = undefined)}
  onconfirm={confirmRemove}
/>
