<script lang="ts">
  interface Props {
    message: string;
    ok: boolean;
  }

  const { message, ok }: Props = $props();
</script>

<div class="size-full bg-base-100" role="img" aria-label="{ok ? 'After' : 'Before'}: {message}">
  <div class="flex gap-1 border-b border-[var(--brand-border)] bg-base-200 px-2 py-1.5" aria-hidden="true">
    {#each [0, 1, 2] as dot (dot)}<span class="size-1.5 rounded-full bg-base-300"></span>{/each}
  </div>
  <div class="flex flex-col gap-2 p-4" aria-hidden="true">
    <div class="rounded px-2 py-1.5 text-center text-xs {ok ? 'bg-primary text-primary-content' : 'bg-error/15 text-error'}">{message}</div>
    <div class="h-1.5 w-12 self-center rounded bg-base-300"></div>
    <div class="h-5 rounded border border-[var(--brand-border)]"></div>
    <div class="h-5 rounded border border-[var(--brand-border)]"></div>
    <div class="h-5 rounded text-center text-[10px] leading-5 {ok ? 'bg-primary text-primary-content' : 'bg-base-300'}">{ok ? 'Sign in' : ''}</div>
  </div>
</div>
