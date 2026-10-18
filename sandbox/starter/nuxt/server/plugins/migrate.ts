import { getMigrations } from 'better-auth/db/migration'
import { auth } from '../infrastructure/auth'

// Creates or updates the login tables when the server starts.
export default defineNitroPlugin(async () => {
  const { runMigrations } = await getMigrations(auth.options)
  await runMigrations()
})
