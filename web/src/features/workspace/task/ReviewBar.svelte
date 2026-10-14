<script lang="ts">
  import { proofCopy } from '../content/tasks-copy';

  interface Props {
    proven: boolean;
    onaccept: () => void;
    onchanges: () => void;
  }

  const { proven, onaccept, onchanges }: Props = $props();
</script>

<div class="mx-auto w-full max-w-[51rem] px-4 pb-3 sm:px-6">
  <div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-primary/40 bg-base-100 px-4 py-2.5">
    <p class="text-[var(--brand-text-secondary)]">
      {proven ? `${proofCopy.verified}. Nothing is final until you accept.` : 'The app could not check this itself. Look over the changes before you accept.'}
    </p>
    <div class="flex gap-2">
      <button class="btn btn-outline btn-sm" type="button" onclick={onchanges}>Ask for changes</button>
      <button class="btn btn-primary btn-sm" type="button" onclick={onaccept}>Accept</button>
    </div>
  </div>
</div>
