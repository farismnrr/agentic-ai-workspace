<script lang="ts">
  import { tick } from 'svelte';
  import Marker from '../../shared/ui/Marker.svelte';
  import { authIllustrations } from './assets/illustrations';
  import AuthLayout from './AuthLayout.svelte';
  import TextField from './components/TextField.svelte';
  import { forgotCopy } from './content';
  import { describeError, requestReset } from './logic/auth-api';
  import { validateEmail } from '../../shared/forms/validation';

  let form: HTMLFormElement | undefined = $state();
  let email = $state('');
  let submitted = $state(false);
  let sent = $state(false);
  let error = $state('');
  let busy = $state(false);

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
      await requestReset(email);
      sent = true;
    } catch (failure) {
      error = describeError(failure);
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>Forgot password · Masih Awam</title></svelte:head>

<AuthLayout illustration={authIllustrations.coatHook} caption={forgotCopy.caption}>
  <h1 class="text-5xl font-medium leading-[1.05] tracking-tight">Lost your <Marker>password?</Marker></h1>
  <p class="mt-4 text-[var(--brand-text-secondary)]">{forgotCopy.subtitle}</p>

  {#if sent}
    <div class="alert alert-success alert-soft mt-8" role="status">{forgotCopy.sent}</div>
  {:else}
    <form bind:this={form} class="mt-8 flex flex-col gap-5" novalidate {onsubmit}>
      <TextField id="email" label="Email" type="email" autocomplete="email" placeholder="you@example.com" bind:value={email} validate={validateEmail} {submitted} />
      <button class="btn btn-primary btn-lg min-h-12 rounded-full" type="submit" disabled={busy}>Send reset link</button>
      {#if error !== ''}
        <div class="alert alert-error alert-soft" role="alert">{error}</div>
      {/if}
    </form>
  {/if}

  <p class="mt-6"><a class="link link-hover font-medium text-primary" href="/login">Back to sign in</a></p>
</AuthLayout>
