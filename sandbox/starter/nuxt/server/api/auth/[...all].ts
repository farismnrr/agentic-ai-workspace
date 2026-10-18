import { auth } from '../../infrastructure/auth'

export default defineEventHandler((event) => auth.handler(toWebRequest(event)))
