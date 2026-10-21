<script lang="ts">
  import FormField from '../components/FormField.svelte';
  import type { CertificateField, TLSChoice } from './database-api';

  interface Props {
    tlsChoice: TLSChoice;
    caSet: boolean; clientCertSet: boolean; clientKeySet: boolean;
    saving: boolean;
    caFileName?: string; clientCertFileName?: string; clientKeyFileName?: string;
    caError?: string; clientCertError?: string; clientKeyError?: string;
    onfile: (field: CertificateField, value: string, fileName: string) => void;
    onremove: (field: 'ca' | 'client') => void;
  }
  let {
    tlsChoice, caSet, clientCertSet, clientKeySet, saving, onfile, onremove,
    caFileName = $bindable(''), clientCertFileName = $bindable(''), clientKeyFileName = $bindable(''),
    caError = $bindable(''), clientCertError = $bindable(''), clientKeyError = $bindable(''),
  }: Props = $props();

  async function readCertificate(event: Event, field: CertificateField): Promise<void> {
    const input = event.currentTarget as HTMLInputElement, file = input.files?.[0];
    if (file === undefined) return;
    setError(field, '');
    if (file.size > 16 * 1024) {
      onfile(field, '', ''); setError(field, 'Choose a file that is 16 KB or smaller.'); input.value = ''; return;
    }
    try {
      const text = await file.text();
      if (!text.includes('-----BEGIN')) throw new Error('This file does not contain a PEM block.');
      onfile(field, text, file.name);
    } catch (error) {
      setError(field, error instanceof Error ? error.message : 'Could not read this file. Choose it again.');
      onfile(field, '', ''); input.value = '';
    }
  }
  function setError(field: CertificateField, value: string): void {
    if (field === 'ca') caError = value; else if (field === 'clientCert') clientCertError = value; else clientKeyError = value;
  }
</script>

{#if tlsChoice !== 'none' || caSet}
  <FormField id="database-ca-cert" label="CA certificate" error={tlsChoice === 'none' ? '' : caError}>
    {#snippet children({ describedby, invalid })}
      <div class="space-y-2">
        {#if caSet}<div class="flex items-center gap-2"><span class="badge badge-sm badge-success badge-soft">Saved</span><button class="btn btn-ghost btn-xs" type="button" aria-label="Remove saved CA certificate" disabled={saving} onclick={() => { onremove('ca'); }}>Remove</button></div>{/if}
        {#if tlsChoice !== 'none'}<input id="database-ca-cert" class="file-input file-input-sm w-full" type="file" accept=".pem,.crt,.cer,.key" aria-describedby={describedby} aria-invalid={invalid} onchange={(event) => void readCertificate(event, 'ca')} />{/if}
        {#if caFileName !== ''}<p class="text-xs" aria-live="polite">{caFileName}</p>{/if}
      </div>
    {/snippet}
  </FormField>
{/if}
{#if tlsChoice === 'full' || clientCertSet || clientKeySet}
  <div class="grid gap-3 sm:grid-cols-2">
    <FormField id="database-client-cert" label="Client certificate" error={tlsChoice === 'full' ? clientCertError : ''}>
      {#snippet children({ describedby, invalid })}
        <div class="space-y-2">
          {#if clientCertSet || clientKeySet}<div class="flex items-center gap-2">{#if clientCertSet}<span class="badge badge-sm badge-success badge-soft">Saved</span>{/if}<button class="btn btn-ghost btn-xs" type="button" aria-label="Remove saved client certificate and key" disabled={saving} onclick={() => { onremove('client'); }}>Remove</button></div>{/if}
          {#if tlsChoice === 'full'}<input id="database-client-cert" class="file-input file-input-sm w-full" type="file" accept=".pem,.crt,.cer,.key" aria-describedby={describedby} aria-invalid={invalid} onchange={(event) => void readCertificate(event, 'clientCert')} />{/if}
          {#if clientCertFileName !== ''}<p class="text-xs" aria-live="polite">{clientCertFileName}</p>{/if}
        </div>
      {/snippet}
    </FormField>
    <FormField id="database-client-key" label="Client key" error={tlsChoice === 'full' ? clientKeyError : ''}>
      {#snippet children({ describedby, invalid })}
        <div class="space-y-2">
          {#if clientKeySet}<span class="badge badge-sm badge-success badge-soft">Saved</span>{/if}
          {#if tlsChoice === 'full'}<input id="database-client-key" class="file-input file-input-sm w-full" type="file" accept=".pem,.crt,.cer,.key" aria-describedby={describedby} aria-invalid={invalid} onchange={(event) => void readCertificate(event, 'clientKey')} />{/if}
          {#if clientKeyFileName !== ''}<p class="text-xs" aria-live="polite">{clientKeyFileName}</p>{/if}
        </div>
      {/snippet}
    </FormField>
  </div>
{/if}
