import { betterAuth } from 'better-auth'
import { db } from './db'

const config = useRuntimeConfig()

export const auth = betterAuth({
  database: db,
  secret: config.authSecret,
  // The app may live under an address prefix (the sites proxy); the login routes live under it too.
  basePath: `${config.app.baseURL.replace(/\/$/, '')}/api/auth`,
  emailAndPassword: { enabled: true, minPasswordLength: 10 },
})
