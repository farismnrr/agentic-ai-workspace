// Every /app page needs a session. The API checks it on the server too; this only sends a visitor to the login.
export default defineNuxtRouteMiddleware(async (to) => {
  if (!to.path.startsWith('/app')) return

  const base = useRuntimeConfig().app.baseURL
  const headers = import.meta.server ? useRequestHeaders(['cookie']) : undefined
  const session = await $fetch<{ user?: unknown } | null>(`${base}api/auth/get-session`, { headers }).catch(() => null)

  if (!session?.user) return navigateTo('/login')
})
