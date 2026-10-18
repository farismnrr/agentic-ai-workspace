<script setup lang="ts">
const base = useRuntimeConfig().app.baseURL
const toast = useToast()
const fields = [
  { name: 'email', type: 'text' as const, label: 'Email', required: true },
  { name: 'password', type: 'password' as const, label: 'Password', required: true },
]

async function signIn(event: { data: { email: string, password: string } }): Promise<void> {
  try {
    await $fetch(`${base}api/auth/sign-in/email`, { method: 'POST', body: event.data })
    await navigateTo('/app')
  }
  catch {
    toast.add({ title: 'Wrong email or password', color: 'error' })
  }
}
</script>

<template>
  <div class="flex min-h-[70vh] items-center justify-center p-4">
    <UPageCard class="w-full max-w-md">
      <UAuthForm title="Sign in" description="Enter your email and password." icon="i-lucide-user" :fields="fields" :submit="{ label: 'Sign in' }" @submit="signIn">
        <template #footer>No account yet? <ULink to="/register" class="text-primary font-medium">Create one</ULink></template>
      </UAuthForm>
    </UPageCard>
  </div>
</template>
