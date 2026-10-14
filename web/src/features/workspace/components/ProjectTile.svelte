<script lang="ts">
  import type { Tone } from '../content/types';

  interface Props {
    name: string;
    tone: Tone;
    class?: string;
  }

  const tones: Record<Tone, string> = {
    primary: 'bg-primary text-primary-content',
    secondary: 'bg-secondary text-secondary-content',
    accent: 'bg-accent text-accent-content',
  };

  const { name, tone, class: className = 'size-11 text-lg' }: Props = $props();
</script>

<span class="grid shrink-0 place-items-center rounded-lg font-medium {tones[tone]} {className}" aria-hidden="true">
  {name.slice(0, 1)}
</span>
