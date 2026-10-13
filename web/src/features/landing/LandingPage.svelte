<script lang="ts">
  import LazySection from '../../shared/ui/LazySection.svelte';
  import HeroSection from './components/HeroSection.svelte';
  import HowItWorksSection from './components/HowItWorksSection.svelte';
  import LandingNav from './components/LandingNav.svelte';
  import WhyDifferentSection from './components/WhyDifferentSection.svelte';
</script>

<LandingNav />
<main>
  <HeroSection />
  <HowItWorksSection />
  <WhyDifferentSection />
  <LazySection load={() => import('./components/FeaturesSection.svelte')} minHeight="40rem" />
  <LazySection id="faq" load={() => import('./components/FaqSection.svelte')} minHeight="32rem" />
  <LazySection id="get-started" load={() => import('./components/CtaSection.svelte')} minHeight="24rem" />
</main>
<LazySection load={() => import('./components/LandingFooter.svelte')} minHeight="10rem" />
