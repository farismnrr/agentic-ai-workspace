<script lang="ts">
  import CtaSection from './components/CtaSection.svelte';
  import FaqSection from './components/FaqSection.svelte';
  import FeaturesSection from './components/FeaturesSection.svelte';
  import HeroSection from './components/HeroSection.svelte';
  import HowItWorksSection from './components/HowItWorksSection.svelte';
  import LandingFooter from './components/LandingFooter.svelte';
  import LandingNav from './components/LandingNav.svelte';
  import WhyDifferentSection from './components/WhyDifferentSection.svelte';
</script>

<LandingNav />
<main class="overflow-x-clip">
  <HeroSection />
  <HowItWorksSection />
  <WhyDifferentSection />
  <FeaturesSection />
  <FaqSection />
  <CtaSection />
</main>
<LandingFooter />
