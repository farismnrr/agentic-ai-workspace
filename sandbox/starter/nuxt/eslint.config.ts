import withNuxt from './.nuxt/eslint.config.mjs'
import security from 'eslint-plugin-security'

export default withNuxt(security.configs.recommended)
