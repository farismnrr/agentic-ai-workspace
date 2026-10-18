import { auth } from '../infrastructure/auth'

// Who is signed in. Every protected API starts the same way: read the session on the server, answer 401 without one.
export default defineEventHandler(async (event) => {
  const session = await auth.api.getSession({ headers: event.headers })
  if (!session) throw createError({ statusCode: 401, statusMessage: 'Sign in first' })

  return { id: session.user.id, name: session.user.name, email: session.user.email }
})
