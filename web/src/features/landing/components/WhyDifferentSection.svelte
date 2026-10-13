<script lang="ts">
  import Illustration from '../../../shared/ui/Illustration.svelte';
  import Marker from '../../../shared/ui/Marker.svelte';
  import { illustrations } from '../assets/illustrations';
</script>

<section id="why-different" class="mx-auto grid w-full max-w-6xl items-center gap-10 px-6 py-14 lg:grid-cols-[0.8fr_1.2fr] lg:py-20">
  <div>
    <p class="text-primary">Why it’s different</p>
    <h2 class="mt-4 font-display text-5xl font-medium leading-[1.05] tracking-tight sm:text-6xl">
      Not a promise.<br />
      A <Marker>receipt.</Marker>
    </h2>
    <p class="mt-6 max-w-sm text-lg leading-relaxed text-[var(--brand-text-secondary)]">
      Other AI tools ask you to trust them. Masih Awam shows you what it did.
    </p>
  </div>

  <figure>
    <Illustration asset={illustrations.receipt} class="h-auto w-full" />
    <figcaption class="mt-3 flex justify-between gap-4 text-sm">
      <span class="text-[var(--brand-text-tertiary)]">What you usually get</span>
      <span class="font-medium text-primary">What you get with Masih Awam</span>
    </figcaption>
  </figure>
</section>
