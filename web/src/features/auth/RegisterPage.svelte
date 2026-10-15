<script lang="ts">
  import { tick } from 'svelte';
  import Icon from '../../shared/ui/Icon.svelte';
  import Marker from '../../shared/ui/Marker.svelte';
  import { authIllustrations } from './assets/illustrations';
  import AuthLayout from './AuthLayout.svelte';
  import PasswordField from '../../shared/forms/PasswordField.svelte';
  import PasswordStrength from './components/PasswordStrength.svelte';
  import TextField from './components/TextField.svelte';
  import { goTo } from '../../app/router';
  import { nextPath } from '../../shared/session/next-path';
  import { setUser } from '../../shared/session/session';
  import { registerCopy } from './content';
  import { createAccount, describeError } from './logic/auth-api';
  import { validateConfirm, validateEmail, validateName, validatePassword } from '../../shared/forms/validation';

  let form: HTMLFormElement | undefined = $state();
  let name = $state('');
  let email = $state('');
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
      setUser(await createAccount(name, email, password));
      goTo(nextPath(window.location.search));
    } catch (failure) {
      error = describeError(failure);
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>Create account · Masih Awam</title></svelte:head>

<AuthLayout illustration={authIllustrations.seedling} caption={registerCopy.caption}>
  <h1 class="text-5xl font-medium leading-[1.05] tracking-tight">Start from <Marker>zero.</Marker></h1>
  <p class="mt-4 text-[var(--brand-text-secondary)]">{registerCopy.subtitle}</p>

  <form bind:this={form} class="mt-8 flex flex-col gap-4" novalidate {onsubmit}>
    <TextField id="name" label="Your name" autocomplete="name" placeholder="Sari" bind:value={name} validate={validateName} {submitted} />
    <TextField id="email" label="Email" type="email" autocomplete="email" placeholder="you@example.com" bind:value={email} validate={validateEmail} {submitted} />
    <PasswordField id="password" label="Password" autocomplete="new-password" placeholder="At least 8 characters" bind:value={password} validate={validatePassword} {submitted}>
      {#snippet below()}<PasswordStrength {password} />{/snippet}
    </PasswordField>
    <PasswordField id="confirm" label="Confirm password" autocomplete="new-password" placeholder="Type it again" bind:value={confirm} validate={(value: string) => validateConfirm(password, value)} {submitted} {matched}>
      {#snippet below()}
        {#if matched}
          <p class="flex items-center gap-1.5 text-sm text-[var(--brand-text-secondary)]"><Icon name="check" class="size-4" />Passwords match</p>
        {/if}
      {/snippet}
    </PasswordField>
    <button class="btn btn-primary btn-lg mt-2 min-h-12 rounded-full" type="submit" disabled={busy}>Create account</button>
    {#if error !== ''}
      <div class="alert alert-error alert-soft" role="alert">{error}</div>
    {/if}
  </form>

  <p class="mt-6">Already have an account? <a class="link link-hover font-medium text-primary" href="/login">Sign in</a></p>
</AuthLayout>
