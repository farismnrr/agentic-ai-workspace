<script setup lang="ts">
const base = useRuntimeConfig().app.baseURL
const items = [[
  { label: 'Dashboard', icon: 'i-lucide-layout-dashboard', to: '/app' },
  // One entry per feature page of the PRD, for example { label: 'Orders', icon: 'i-lucide-shopping-cart', to: '/app/orders' }
]]

async function signOut(): Promise<void> {
  await $fetch(`${base}api/auth/sign-out`, { method: 'POST' })
  await navigateTo('/login')
}
</script>

<template>
  <UDashboardGroup>
    <UDashboardSidebar collapsible resizable>
      <template #header>{{ brand.name }}</template>
      <UNavigationMenu :items="items" orientation="vertical" />
      <template #footer>
        <UButton color="neutral" variant="ghost" icon="i-lucide-log-out" label="Sign out" @click="signOut" />
      </template>
    </UDashboardSidebar>
    <UDashboardPanel>
      <template #header><UDashboardNavbar :title="brand.name" /></template>
      <template #body><slot /></template>
    </UDashboardPanel>
  </UDashboardGroup>
</template>
