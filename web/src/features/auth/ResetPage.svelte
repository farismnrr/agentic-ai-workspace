<script lang="ts">
  import { tick } from 'svelte';
  import Icon from '../../shared/ui/Icon.svelte';
  import Marker from '../../shared/ui/Marker.svelte';
  import { navigate } from '../../app/router';
  import { authIllustrations } from './assets/illustrations';
  import AuthLayout from './AuthLayout.svelte';
  import PasswordField from '../../shared/forms/PasswordField.svelte';
  import PasswordStrength from './components/PasswordStrength.svelte';
  import { resetCopy } from './content';
  import { describeError, resetPassword } from './logic/auth-api';
  import { validateConfirm, validatePassword } from '../../shared/forms/validation';

  const token = new URLSearchParams(window.location.search).get('token') ?? '';

  let form: HTMLFormElement | undefined = $state();
  let password = $state('');
  let confirm = $state('');
  let submitted = $state(false);
  let error = $state('');
  let busy = $state(false);

  const matched = $derived(confirm !== '' && confirm === password);

  async function onsubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    submitted = true;
    error = '';
    await tick();
    const invalid = form?.querySelector<HTMLElement>('[aria-invalid="true"]');
    if (invalid) {
      invalid.focus();
      return;
    }
    busy = true;
    try {
      await resetPassword(token, password);
      navigate('/login');
    } catch (failure) {
      error = describeError(failure);
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>Choose a new password · Masih Awam</title></svelte:head>

<AuthLayout illustration={authIllustrations.seedling} caption={resetCopy.caption}>
  <h1 class="text-5xl font-medium leading-[1.05] tracking-tight">New <Marker>password.</Marker></h1>
  <p class="mt-4 text-[var(--brand-text-secondary)]">{resetCopy.subtitle}</p>

  {#if token === ''}
    <div class="alert alert-warning alert-soft mt-8" role="alert">{resetCopy.missing}</div>
    <p class="mt-6"><a class="link link-hover font-medium text-primary" href="/forgot">Get a new link</a></p>
  {:else}
    <form bind:this={form} class="mt-8 flex flex-col gap-4" novalidate {onsubmit}>
      <PasswordField id="password" label="New password" autocomplete="new-password" placeholder="At least 8 characters" bind:value={password} validate={validatePassword} {submitted}>
        {#snippet below()}<PasswordStrength {password} />{/snippet}
      </PasswordField>
      <PasswordField id="confirm" label="Confirm password" autocomplete="new-password" placeholder="Type it again" bind:value={confirm} validate={(value: string) => validateConfirm(password, value)} {submitted} {matched}>
        {#snippet below()}
          {#if matched}
            <p class="flex items-center gap-1.5 text-sm text-[var(--brand-text-secondary)]"><Icon name="check" class="size-4" />Passwords match</p>
          {/if}
        {/snippet}
      </PasswordField>
      <button class="btn btn-primary btn-lg mt-2 min-h-12 rounded-full" type="submit" disabled={busy}>Save new password</button>
      {#if error !== ''}
        <div class="alert alert-error alert-soft" role="alert">{error}</div>
      {/if}
    </form>
  {/if}
</AuthLayout>
