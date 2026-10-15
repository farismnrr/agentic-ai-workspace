<script lang="ts">
  import { reveal } from '../../../shared/motion/reveal';
  import Illustration from '../../../shared/ui/Illustration.svelte';
  import Marker from '../../../shared/ui/Marker.svelte';
  import { illustrations } from '../assets/illustrations';
</script>

<section id="why-different" class="mx-auto grid grid-cols-1 w-full max-w-6xl items-center gap-10 px-6 py-14 lg:grid-cols-[0.8fr_1.2fr] lg:py-20">
  <div use:reveal>
    <p class="text-primary">Why it’s different</p>
    <h2 class="mt-4 text-4xl font-medium leading-[1.05] tracking-tight sm:text-5xl lg:text-6xl">
      Not a promise.<br />
      A <Marker>receipt.</Marker>
    </h2>
    <p class="mt-6 max-w-sm text-lg leading-relaxed text-[var(--brand-text-secondary)]">
      Other AI tools ask you to trust them. Masih Awam shows you what it did.
    </p>
  </div>

  <figure use:reveal={120}>
    <Illustration asset={illustrations.receipt} class="h-auto w-full" />
  </figure>
</section>
