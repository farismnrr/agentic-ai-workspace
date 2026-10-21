<script lang="ts">
  import { tick } from 'svelte';
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import ConfirmModal from '../../components/ConfirmModal.svelte';
  import { deleteVariable, listVariables, saveVariable, getDatabase, type EnvVariable } from './env-api';
  import { formatUpdatedAt, normalizeVariableName, variableErrorMessage, variableValue } from './env-logic';

  interface Props { projectId: string; revision: number }
  const { projectId, revision }: Props = $props();
  let variables = $state<EnvVariable[]>([]);
  let managedName = $state('');
  let name = $state('');
  let value = $state('');
  let isSecret = $state(true);
  let editing = $state(false);
  let message = $state('');
  let error = $state(false);
  let busy = $state(false);
  let loading = $state(true);
  let valueInput = $state<HTMLInputElement>();
  let pending = $state<{ kind: 'replace' | 'delete'; item: EnvVariable }>();

  $effect(() => {
    if (projectId !== '' && revision >= 0) reload().catch(() => undefined);
  });

  async function reload(): Promise<void> {
    loading = true;
    try {
      const [items, database] = await Promise.all([listVariables(projectId), getDatabase(projectId)]);
      variables = items;
      managedName = database.configured ? database.envName : '';
      error = false;
    } catch {
      message = 'Could not load environment variables.';
      error = true;
    } finally { loading = false; }
  }

  async function beginReplace(item: EnvVariable): Promise<void> {
    name = item.name;
    value = '';
    isSecret = item.isSecret;
    editing = true;
    message = 'Enter a new value. The saved value cannot be viewed.';
    error = false;
    await tick();
    valueInput?.focus();
  }

  async function submit(): Promise<void> {
    const nextName = normalizeVariableName(name.trim());
    if (value === '') { message = 'Enter a value to save.'; error = true; return; }
    if (nextName === managedName && !editing) {
      message = 'This variable is managed in the Database section.'; error = true; return;
    }
    const existing = variables.find((item) => item.name === nextName);
    if (existing && !editing) { pending = { kind: 'replace', item: existing }; return; }
    await save(nextName);
  }

  async function save(nextName: string): Promise<void> {
    busy = true;
    try {
      await saveVariable(projectId, { name: nextName, value, isSecret });
      name = ''; value = ''; isSecret = true; editing = false;
      message = 'Variable saved.'; error = false;
      await reload();
    } catch (cause) { message = variableErrorMessage(cause); error = true; }
    finally { busy = false; pending = undefined; }
  }

  async function confirmAction(): Promise<void> {
    if (pending === undefined) return;
    if (pending.kind === 'replace') { await save(pending.item.name); return; }
    try {
      await deleteVariable(projectId, pending.item.name);
      message = 'Variable deleted.'; error = false;
      await reload();
    } catch { message = 'Could not delete this variable.'; error = true; }
    finally { pending = undefined; }
  }
</script>

<section class="mb-8" aria-labelledby="variables-heading">
  <div class="mb-1"><h2 id="variables-heading" class="text-sm font-semibold">Variables</h2><p class="mt-0.5 text-xs text-[var(--brand-text-tertiary)]">Secret values are masked and never shown again after saving.</p></div>
  <div class="divide-y divide-[var(--brand-border)]">
  {#if loading}
    <p class="py-3" aria-busy="true"><span class="loading loading-spinner loading-sm" aria-label="Loading variables"></span></p>
  {:else if variables.length === 0}
    <EmptyState compact icon="key" title="No variables yet" body="Add a variable below for runs in this project." />
  {:else}
    <ul class="divide-y divide-[var(--brand-border)]">
      {#each variables as item (item.name)}
        <li class="flex flex-wrap items-center gap-3 py-3">
          <div class="min-w-0 flex-1">
            <p class="font-mono text-sm font-medium">{item.name}</p>
            <p class="break-all text-sm text-[var(--brand-text-secondary)]">{variableValue(item)}</p>
            {#if item.name === managedName}<p class="mt-1 text-xs text-[var(--brand-text-secondary)]">Managed in the Database section.</p>{/if}
          </div>
          <div class="flex items-center gap-2 text-xs text-[var(--brand-text-secondary)]">
            {#if item.isSecret}<span class="badge badge-outline badge-sm">Secret</span>{/if}
            <time datetime={item.updatedAt}>{formatUpdatedAt(item.updatedAt)}</time>
          </div>
          {#if item.name !== managedName}
            <button class="btn btn-ghost btn-sm min-h-11" type="button" aria-label="Replace {item.name}" onclick={() => void beginReplace(item)}>
              <Icon name="pencil" class="size-4" />Replace
            </button>
            <button class="btn btn-ghost btn-sm min-h-11 text-error" type="button" aria-label="Delete {item.name}" onclick={() => (pending = { kind: 'delete', item })}>
              <Icon name="trash" class="size-4" />Delete
            </button>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
  <form class="grid gap-3 py-4" onsubmit={(event) => { event.preventDefault(); void submit(); }}>
    <div class="grid gap-3 sm:grid-cols-2">
      <label>
        <span class="label-text mb-1 block">Name</span>
        <input class="input input-sm w-full font-mono uppercase" value={name} readonly={editing}
          oninput={(event) => (name = normalizeVariableName(event.currentTarget.value))}
          placeholder="SERVICE_TOKEN" required maxlength="64" pattern="[A-Za-z_][A-Za-z0-9_]*" autocomplete="off" />
        <span class="mt-1 block text-xs text-[var(--brand-text-secondary)]">Uppercase letters, numbers and underscores.</span>
      </label>
      <label>
        <span class="label-text mb-1 block">Value</span>
        <input bind:this={valueInput} class="input input-sm w-full" type="password" bind:value required
          maxlength="4096" autocomplete="new-password" placeholder="Enter value" />
      </label>
    </div>
    <label class="flex w-fit items-center gap-2 text-sm"><input class="checkbox checkbox-sm" type="checkbox" bind:checked={isSecret} />Secret</label>
    <div class="flex flex-wrap items-center gap-3">
      <button class="btn btn-primary btn-sm rounded-lg min-h-11" type="submit" disabled={busy}>
        {editing ? 'Save replacement' : 'Add variable'}
      </button>
      {#if editing}<button class="btn btn-ghost btn-sm min-h-11" type="button" onclick={() => { editing = false; name = ''; value = ''; }}>Cancel</button>{/if}
    </div>
    <div class="min-h-6" aria-live="polite">
      {#if message !== ''}<p class="flex items-center gap-2 text-sm" class:text-error={error}><Icon name={error ? 'alert-triangle' : 'check-circle'} class="size-4 shrink-0" />{message}</p>{/if}
    </div>
  </form>
  </div>
</section>

<ConfirmModal
  open={pending !== undefined}
  title={pending?.kind === 'replace' ? `Replace ${pending.item.name}?` : `Delete ${pending?.item.name ?? ''}?`}
  body={pending?.kind === 'replace' ? 'The saved value will be replaced and cannot be recovered.' : 'This variable will be removed from project runs.'}
  confirmLabel={pending?.kind === 'replace' ? 'Replace variable' : 'Delete variable'}
  cancelLabel="Cancel"
  oncancel={() => (pending = undefined)}
  onconfirm={confirmAction}
/>
