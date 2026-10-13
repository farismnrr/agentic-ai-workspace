<script lang="ts">
  import { tick } from 'svelte';
  import Marker from '../../shared/ui/Marker.svelte';
  import { authIllustrations } from './assets/illustrations';
  import AuthLayout from './AuthLayout.svelte';
  import PasswordField from './components/PasswordField.svelte';
  import TextField from './components/TextField.svelte';
  import { loginCopy, NOT_CONNECTED_MESSAGE } from './content';
  import { validateEmail } from './validation';

  let form: HTMLFormElement | undefined = $state();
  let email = $state('');
  let password = $state('');
  let submitted = $state(false);
  let notice = $state(false);

  const validatePassword = (value: string): string => (value === '' ? 'Enter your password.' : '');

  async function onsubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    submitted = true;
    notice = false;
    await tick();
    const invalid = form?.querySelector<HTMLElement>('[aria-invalid="true"]');
    if (invalid) {
      invalid.focus();
      return;
    }
    notice = true;
  }
</script>

<svelte:head><title>Sign in · Masih Awam</title></svelte:head>

<AuthLayout illustration={authIllustrations.coatHook} caption={loginCopy.caption}>
  <h1 class="text-5xl font-medium leading-[1.05] tracking-tight">Welcome <Marker>back.</Marker></h1>
  <p class="mt-4 text-[var(--brand-text-secondary)]">{loginCopy.subtitle}</p>

  <form bind:this={form} class="mt-8 flex flex-col gap-5" novalidate {onsubmit}>
    <TextField id="email" label="Email" type="email" autocomplete="email" placeholder="you@example.com" bind:value={email} validate={validateEmail} {submitted} />
    <PasswordField id="password" label="Password" autocomplete="current-password" placeholder="Your password" bind:value={password} validate={validatePassword} {submitted} />
    <button class="btn btn-primary btn-lg min-h-12 rounded-full" type="submit">Sign in</button>
    {#if notice}
      <div class="alert alert-info alert-soft" role="alert">{NOT_CONNECTED_MESSAGE}</div>
    {/if}
  </form>

  <p class="mt-6 text-[var(--brand-text-secondary)]">New here? <a class="link link-hover font-medium text-primary" href="/register">Create an account</a></p>
</AuthLayout>
