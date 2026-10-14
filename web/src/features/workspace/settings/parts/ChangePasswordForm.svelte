<script lang="ts">
  import PasswordField from '../../../../shared/forms/PasswordField.svelte';
  import { validatePassword } from '../../../../shared/forms/validation';
  import { changePassword, describeAccountError } from '../../logic/account-api';

  let current = $state('');
  let next = $state('');
  let submitted = $state(false);
  let busy = $state(false);
  let error = $state('');
  let done = $state(false);

  const requireCurrent = (value: string): string => (value === '' ? 'Enter your current password.' : '');

  async function onsubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    submitted = true;
    done = false;
    error = '';
    if (requireCurrent(current) !== '' || validatePassword(next) !== '') return;
    busy = true;
    try {
      await changePassword(current, next);
      done = true;
      current = '';
      next = '';
      submitted = false;
    } catch (failure) {
      error = describeAccountError(failure);
    } finally {
      busy = false;
    }
  }
</script>

<form class="flex max-w-sm flex-col gap-3 py-3" novalidate {onsubmit}>
  <PasswordField id="current-password" label="Current password" autocomplete="current-password" bind:value={current} validate={requireCurrent} {submitted} />
  <PasswordField id="new-password" label="New password" autocomplete="new-password" placeholder="At least 8 characters" bind:value={next} validate={validatePassword} {submitted} />
  <div class="flex items-center gap-3">
    <button class="btn btn-sm btn-primary" type="submit" disabled={busy}>Change password</button>
    {#if done}<span class="text-sm text-success" role="status">Saved. Other devices were signed out.</span>{/if}
  </div>
  {#if error !== ''}<div class="alert alert-error alert-soft text-sm" role="alert">{error}</div>{/if}
</form>
