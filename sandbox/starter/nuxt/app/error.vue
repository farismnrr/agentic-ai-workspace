<script setup lang="ts">
defineProps<{ error: { statusCode: number } }>()
</script>

<template>
  <UApp>
    <div class="flex min-h-screen items-center justify-center p-4">
      <UEmpty
        icon="i-lucide-compass"
        :title="error.statusCode === 404 ? 'Page not found' : 'Something went wrong'"
        description="Check the address, or go back to the start."
        :actions="[{ label: 'Go home', to: '/', onClick: () => clearError() }]"
      />
    </div>
  </UApp>
</template>
