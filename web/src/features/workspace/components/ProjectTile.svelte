<script lang="ts">
  interface Props {
    name: string;
    id?: string | undefined;
    class?: string;
  }

  const tones = ['bg-primary text-primary-content', 'bg-secondary text-secondary-content', 'bg-accent text-accent-content'];

  const { name, id = '', class: className = 'size-11 text-lg' }: Props = $props();
  const seed = $derived(id === '' ? name : id);
  const tone = $derived(tones[hash(seed) % tones.length] ?? '');

  function hash(text: string): number {
    let sum = 0;
    for (let i = 0; i < text.length; i++) sum += text.charCodeAt(i);
    return sum;
  }
</script>

<span class="grid shrink-0 place-items-center rounded-lg font-medium {tone} {className}" aria-hidden="true">
  {name.slice(0, 1).toUpperCase()}
</span>
