<script lang="ts">
  import { onMount } from 'svelte';
  import { authIllustrations } from './assets/illustrations';
  import AuthLayout from './AuthLayout.svelte';
  import { consentCopy } from './content';
  import { decide, loadConsent, type ConsentRequest } from './logic/consent-api';

  const requestId = new URLSearchParams(window.location.search).get('request') ?? '';

  let consent: ConsentRequest | undefined = $state();
  let failed = $state(false);
  let busy = $state(false);

  onMount(async () => {
    try {
      consent = await loadConsent(requestId);
    } catch {
      failed = true;
    }
  });

  async function choose(approve: boolean): Promise<void> {
    busy = true;
    try {
      window.location.assign(await decide(requestId, approve));
    } catch {
      failed = true;
      busy = false;
    }
  }
</script>

<svelte:head><title>Connect an app · Masih Awam</title></svelte:head>

<AuthLayout illustration={authIllustrations.coatHook} caption={consentCopy.caption}>
  {#if failed}
    <h1 class="text-4xl font-medium leading-tight tracking-tight">{consentCopy.expiredTitle}</h1>
    <p class="mt-4 text-[var(--brand-text-secondary)]">{consentCopy.expiredBody}</p>
    <a class="btn btn-primary mt-8 rounded-full" href="/app">{consentCopy.back}</a>
  {:else if consent === undefined}
    <span class="loading loading-spinner" aria-label="Loading"></span>
  {:else}
    <h1 class="text-4xl font-medium leading-tight tracking-tight">{consent.clientName} wants to connect</h1>
    <p class="mt-4 text-[var(--brand-text-secondary)]">
      You will be sent back to <b class="text-base-content">{consent.redirectHost}</b> afterwards. Only continue if you started this yourself.
    </p>
    <h2 class="mt-8 text-sm font-semibold">{consentCopy.canDo}</h2>
    <ul class="mt-2 divide-y divide-[var(--brand-border)] border-y border-[var(--brand-border)]">
      {#each consent.scopes as scope (scope.id)}<li class="py-3">{scope.description}</li>{/each}
    </ul>
    <div class="mt-8 flex gap-3">
      <button class="btn btn-primary min-h-12 flex-1 rounded-full" type="button" disabled={busy} onclick={() => choose(true)}>Allow</button>
      <button class="btn min-h-12 flex-1 rounded-full" type="button" disabled={busy} onclick={() => choose(false)}>Deny</button>
    </div>
    <p class="mt-4 text-sm text-[var(--brand-text-secondary)]">{consentCopy.revoke}</p>
  {/if}
</AuthLayout>
