import { timingSafeEqual } from 'node:crypto'

// Gates the whole site behind a single shared password (HTTP Basic Auth -
// username is accepted but not checked, this isn't a multi-user account
// system). Mirrors cmd/api's basicAuthMiddleware so both origins are
// protected the same way. Empty/unset APP_PASSWORD disables the check, so
// local dev isn't gated by default.
export default defineEventHandler((event) => {
  const password = useRuntimeConfig().appPassword
  if (!password) return

  const header = getRequestHeader(event, 'authorization') ?? ''
  const [scheme, encoded] = header.split(' ')
  const provided = scheme === 'Basic' && encoded ? Buffer.from(encoded, 'base64').toString('utf-8').split(':')[1] ?? '' : ''

  const providedBuf = Buffer.from(provided)
  const passwordBuf = Buffer.from(password)
  const matches = providedBuf.length === passwordBuf.length && timingSafeEqual(providedBuf, passwordBuf)

  if (!matches) {
    setResponseHeader(event, 'WWW-Authenticate', 'Basic realm="snoopy"')
    throw createError({ statusCode: 401, statusMessage: 'Unauthorized' })
  }
})
