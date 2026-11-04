<script lang="ts">
  import { onDestroy, tick, untrack } from 'svelte';
  import EmptyState from '../../../../shared/ui/EmptyState.svelte';
  import Icon from '../../../../shared/ui/Icon.svelte';
  import ConfirmModal from '../../components/ConfirmModal.svelte';
  import SettingsSection from '../../settings/parts/SettingsSection.svelte';
  import RevealSecretModal from './RevealSecretModal.svelte';
  import VariableForm from './VariableForm.svelte';
  import VariableRow from './VariableRow.svelte';
  import { createRevealState } from './env-reveal.svelte';
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
  let loadFailed = $state(false);
  let variableForm = $state<ReturnType<typeof VariableForm>>();
  let pending = $state<{ kind: 'replace' | 'delete'; item: EnvVariable }>();
  const reveal = createRevealState(() => projectId);

  $effect(() => {
    const activeProjectID = projectId;
    untrack(() => {
      void activeProjectID;
      reveal.closeReveal();
    });
  });

  $effect(() => {
    if (projectId !== '' && revision >= 0) reload().catch(() => undefined);
  });

  function onVisibilityChange(): void {
    if (document.hidden) reveal.closeReveal();
  }

  if (typeof document !== 'undefined') document.addEventListener('visibilitychange', onVisibilityChange);
  onDestroy(() => {
    reveal.closeReveal();
    if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisibilityChange);
  });

  async function reload(): Promise<void> {
    loading = true;
    try {
      const [items, database] = await Promise.all([listVariables(projectId), getDatabase(projectId)]);
      variables = items;
      managedName = database.configured ? database.envName : '';
      loadFailed = false;
      error = false;
    } catch {
      message = 'Could not load environment variables.';
      loadFailed = true;
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
    variableForm?.focusValue();
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

<SettingsSection title="Variables" description="Secret values are masked and never shown again after saving.">
  {#if loading}
    <div class="py-3" role="status"><span class="loading loading-spinner loading-sm" aria-label="Loading variables"></span></div>
  {:else if loadFailed}
    <div class="alert alert-error alert-soft" role="alert">{message}</div>
  {:else if variables.length === 0}
    <EmptyState compact icon="key" title="No variables yet" body="Add a variable below for runs in this project." />
  {:else}
    <ul class="divide-y divide-[var(--brand-border)]">
      {#each variables as item (item.name)}
        <VariableRow
          {item}
          {managedName}
          revealed={reveal.revealed}
          onreveal={(target) => reveal.beginReveal(target)}
          onreplace={(target) => void beginReplace(target)}
          ondelete={(target) => (pending = { kind: 'delete', item: target })}
        />
      {/each}
    </ul>
  {/if}
  {#if !loading && !loadFailed}
    <VariableForm
      bind:this={variableForm}
      bind:name
      bind:value
      bind:isSecret
      bind:editing
      {busy}
      {message}
      {error}
      onsubmit={() => void submit()}
      oncancel={() => { editing = false; name = ''; value = ''; }}
    />
  {/if}
</SettingsSection>

{#if reveal.target !== undefined}
  <RevealSecretModal
    target={reveal.target}
    bind:password={reveal.password}
    message={reveal.message}
    revealing={reveal.revealing}
    oncancel={reveal.closeReveal}
    onsubmit={() => void reveal.submitReveal()}
  />
{/if}
{#if reveal.revealed}
  <div class="sr-only" role="status" aria-live="polite">Secret {reveal.revealed.name} is visible for up to 20 seconds.</div>
{/if}

<ConfirmModal
  open={pending !== undefined}
  title={pending?.kind === 'replace' ? `Replace ${pending.item.name}?` : `Delete ${pending?.item.name ?? ''}?`}
  body={pending?.kind === 'replace' ? 'The saved value will be replaced and cannot be recovered.' : 'This variable will be removed from project runs.'}
  confirmLabel={pending?.kind === 'replace' ? 'Replace variable' : 'Delete variable'}
  cancelLabel="Cancel"
  oncancel={() => (pending = undefined)}
  onconfirm={confirmAction}
/>
