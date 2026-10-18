<script setup lang="ts">
definePageMeta({ layout: 'app' })
const { data: me } = await useFetch<{ name: string }>('api/me', { baseURL: useRuntimeConfig().app.baseURL })

// The PRD's key numbers go here: a label, the real figure from the database, and the change. Add the chart and the
// recent activity under it. While there is no data, the empty state below says what to do first.
const stats: { label: string, value: string, change?: string }[] = []
</script>

<template>
  <div class="space-y-6">
    <h1 class="text-xl font-semibold">Welcome, {{ me?.name }}</h1>
    <div v-if="stats.length" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <UCard v-for="stat in stats" :key="stat.label">
        <p class="text-sm text-muted">{{ stat.label }}</p>
        <p class="mt-1 text-2xl font-semibold">{{ stat.value }}</p>
        <UBadge v-if="stat.change" color="success" variant="subtle" :label="stat.change" />
      </UCard>
    </div>
    <UEmpty v-else icon="i-lucide-inbox" title="Nothing here yet" description="Add your first item to see your numbers appear here." />
  </div>
</template>
