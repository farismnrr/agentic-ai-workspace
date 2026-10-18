<script setup lang="ts">
const links = [
  { label: 'Features', to: '#features' },
  { label: 'Pricing', to: '#pricing' },
]
</script>

<template>
  <div>
    <UHeader :title="brand.name">
      <UNavigationMenu :items="links" />
      <template #right>
        <UButton to="/login" color="neutral" variant="ghost" label="Sign in" />
        <UButton to="/register" :label="brand.cta" />
      </template>
    </UHeader>
    <UMain><slot /></UMain>
    <UFooter>
      <template #left>© {{ brand.name }}</template>
    </UFooter>
  </div>
</template>
