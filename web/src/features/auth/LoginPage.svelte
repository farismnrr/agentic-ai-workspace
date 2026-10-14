<script lang="ts">
  import { tick } from 'svelte';
  import Marker from '../../shared/ui/Marker.svelte';
  import { authIllustrations } from './assets/illustrations';
  import AuthLayout from './AuthLayout.svelte';
  import PasswordField from '../../shared/forms/PasswordField.svelte';
  import TextField from './components/TextField.svelte';
  import { navigate } from '../../app/router';
  import { setUser } from '../../shared/session/session';
  import { loginCopy } from './content';
  import { describeError, nextPath, signIn } from './logic/auth-api';
  import { validateEmail } from '../../shared/forms/validation';

  let form: HTMLFormElement | undefined = $state();
  let email = $state('');
  let password = $state('');
  let submitted = $state(false);
  let error = $state('');
  let busy = $state(false);

  const validatePassword = (value: string): string => (value === '' ? 'Enter your password.' : '');

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
      setUser(await signIn(email, password));
      navigate(nextPath(window.location.search));
    } catch (failure) {
      error = describeError(failure);
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>Sign in · Masih Awam</title></svelte:head>

<AuthLayout illustration={authIllustrations.coatHook} caption={loginCopy.caption}>
  <h1 class="text-5xl font-medium leading-[1.05] tracking-tight">Welcome <Marker>back.</Marker></h1>
  <p class="mt-4 text-[var(--brand-text-secondary)]">{loginCopy.subtitle}</p>

  <form bind:this={form} class="mt-8 flex flex-col gap-5" novalidate {onsubmit}>
    <TextField id="email" label="Email" type="email" autocomplete="email" placeholder="you@example.com" bind:value={email} validate={validateEmail} {submitted} />
    <PasswordField id="password" label="Password" autocomplete="current-password" placeholder="Your password" bind:value={password} validate={validatePassword} {submitted}>
      {#snippet labelAside()}<a class="link link-hover text-sm text-primary" href="/forgot">Forgot your password?</a>{/snippet}
    </PasswordField>
    <button class="btn btn-primary btn-lg min-h-12 rounded-full" type="submit" disabled={busy}>Sign in</button>
    {#if error !== ''}
      <div class="alert alert-error alert-soft" role="alert">{error}</div>
    {/if}
  </form>

  <p class="mt-6 text-[var(--brand-text-secondary)]">New here? <a class="link link-hover font-medium text-primary" href="/register">Create an account</a></p>
</AuthLayout>
