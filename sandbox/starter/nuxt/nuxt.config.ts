export default defineNuxtConfig({
  compatibilityDate: 'latest',
  modules: ['@nuxt/ui', 'nuxt-security', '@nuxt/eslint'],
  css: ['~/assets/css/main.css'],
  runtimeConfig: { authSecret: '' }, // filled by NUXT_AUTH_SECRET
  nitro: { preset: 'bun', typescript: { tsConfig: { compilerOptions: { types: ['bun'] } } } },
  security: { rateLimiter: { tokensPerInterval: 150, interval: 300000 } },
  routeRules: { '/api/auth/**': { security: { rateLimiter: { tokensPerInterval: 10, interval: 60000 } } } },
})
