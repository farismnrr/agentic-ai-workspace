<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';
  import Notice from '../../components/Notice.svelte';

  interface Props {
    text: string;
    options: string[];
    answered?: string | undefined;
  }

  const { text, options, answered }: Props = $props();
  let picked = $state<string | undefined>(undefined);
  let typed = $state('');
  const choice = $derived(answered ?? picked);

  function submit(event: SubmitEvent): void {
    event.preventDefault();
    if (typed.trim() !== '') picked = typed.trim();
  }
</script>

<section class="card border border-warning/50 bg-base-100 p-3" aria-label="Question for you">
  <p class="flex items-start gap-2 font-medium"><Icon name="help-circle" class="mt-0.5 size-5 shrink-0 text-warning" />{text}</p>
  <ul class="mt-3 flex flex-wrap gap-2">
    {#each options as option (option)}
      <li><button class="btn btn-sm rounded-lg {choice === option ? 'btn-primary' : 'btn-outline'}" type="button" disabled={answered !== undefined} onclick={() => (picked = option)}>{option}</button></li>
    {/each}
  </ul>
  {#if answered === undefined}
    <form class="mt-3 flex gap-2" onsubmit={submit}>
      <input class="input input-sm h-10 flex-1" type="text" placeholder="Or type your own answer" aria-label="Your own answer" bind:value={typed} />
      <button class="btn btn-sm rounded-lg" type="submit">Send</button>
    </form>
  {/if}
  {#if picked !== undefined}<div class="mt-3"><Notice show /></div>{/if}
</section>
