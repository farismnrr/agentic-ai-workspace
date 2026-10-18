<script setup lang="ts">
// Replace the copy, the features and the plans with what the PRD says. Keep the order of the sections.
const features = [
  { title: 'First benefit', description: 'What the person gets, in plain words.', icon: 'i-lucide-zap' },
  { title: 'Second benefit', description: 'What the person gets, in plain words.', icon: 'i-lucide-shield-check' },
  { title: 'Third benefit', description: 'What the person gets, in plain words.', icon: 'i-lucide-chart-line' },
]
const plans = [
  { title: 'Free', price: '$0', description: 'To try it', features: ['What is included'], button: { label: 'Start', to: '/register' } },
  { title: 'Pro', price: '$12', billingCycle: '/month', description: 'For regular use', highlight: true, features: ['Everything in Free', 'What is extra'], button: { label: 'Choose Pro', to: '/register' } },
]
</script>

<template>
  <div>
    <UPageHero
      :title="brand.tagline"
      :description="brand.description"
      :links="[{ label: brand.cta, to: '/register', size: 'xl' }, { label: 'See features', to: '#features', color: 'neutral', variant: 'subtle', size: 'xl' }]"
    />
    <UPageSection id="features" title="Everything you need" :features="features" />
    <UPageSection id="pricing" title="Simple pricing"><UPricingPlans :plans="plans" /></UPageSection>
    <UPageCTA title="Ready to start?" :links="[{ label: brand.cta, to: '/register' }]" />
  </div>
</template>
