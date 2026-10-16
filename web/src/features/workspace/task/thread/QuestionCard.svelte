<script lang="ts">
  import Icon from '../../../../shared/ui/Icon.svelte';

  interface Props {
    text: string;
    options?: string[] | undefined;
    answered?: string | undefined;
    /** The teammate asking, when it is not the lead. */
    agent?: string | undefined;
    onanswer: (choice: string) => void;
  }

  const { text, options = [], answered, agent, onanswer }: Props = $props();
  let typed = $state('');

  function submit(event: SubmitEvent): void {
    event.preventDefault();
    if (typed.trim() !== '') onanswer(typed.trim());
  }
</script>

<section class="card border border-warning/50 bg-base-100 p-3" aria-label="Question for you">
  <p class="flex items-start gap-2 font-medium"><Icon name="help-circle" class="mt-0.5 size-5 shrink-0 text-warning" />{#if agent !== undefined}<span class="badge badge-outline badge-sm mr-1.5 align-middle">{agent}</span>{/if}{text}</p>
  {#if answered !== undefined}
    <p class="mt-3 text-sm font-medium">You answered: {answered}</p>
  {:else}
    <ul class="mt-3 flex flex-wrap gap-2">
      {#each options as option (option)}
        <li><button class="btn btn-outline btn-sm rounded-lg" type="button" onclick={() => { onanswer(option); }}>{option}</button></li>
      {/each}
    </ul>
    <form class="mt-3 flex gap-2" onsubmit={submit}>
      <input class="input input-sm h-10 flex-1" type="text" placeholder="Or type your own answer" aria-label="Your own answer" bind:value={typed} />
      <button class="btn btn-sm rounded-lg" type="submit">Send</button>
    </form>
  {/if}
</section>
