<script setup lang="ts">
const base = useRuntimeConfig().app.baseURL
const toast = useToast()
const fields = [
  { name: 'name', type: 'text' as const, label: 'Name', required: true },
  { name: 'email', type: 'text' as const, label: 'Email', required: true },
  { name: 'password', type: 'password' as const, label: 'Password (at least 10 characters)', required: true },
]

async function signUp(event: { data: { name: string, email: string, password: string } }): Promise<void> {
  try {
    await $fetch(`${base}api/auth/sign-up/email`, { method: 'POST', body: event.data })
    await navigateTo('/app')
  }
  catch {
    toast.add({ title: 'Could not create the account', description: 'Use a new email and a password of at least 10 characters.', color: 'error' })
  }
}
</script>

<template>
  <div class="flex min-h-[70vh] items-center justify-center p-4">
    <UPageCard class="w-full max-w-md">
      <UAuthForm title="Create your account" description="It takes a minute." icon="i-lucide-user-plus" :fields="fields" :submit="{ label: 'Create account' }" @submit="signUp">
        <template #footer>Already have an account? <ULink to="/login" class="text-primary font-medium">Sign in</ULink></template>
      </UAuthForm>
    </UPageCard>
  </div>
</template>
