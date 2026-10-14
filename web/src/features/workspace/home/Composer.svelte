<script lang="ts">
  import ComposerBox from '../components/ComposerBox.svelte';
  import { suggestions } from '../content/user';

  let text = $state('');
</script>

<ComposerBox id="composer" placeholder="Describe what you need in plain words…" bind:text withProject />
<ul class="mt-3 flex flex-wrap gap-2">
  {#each suggestions as suggestion (suggestion)}
    <li><button class="btn btn-sm rounded-full border-[var(--brand-border)] bg-base-100 font-normal" type="button" onclick={() => (text = suggestion)}>{suggestion}</button></li>
  {/each}
</ul>
