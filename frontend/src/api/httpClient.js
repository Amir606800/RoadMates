export class APIError extends Error {
  constructor(status, body = {}) {
    super(body.message || 'The request failed. Please try again.')
    this.name = 'APIError'
    this.status = status
    this.code = body.code
    this.requestId = body.request_id
  }
}

// The provider owns the token; this client reads its in-memory ref on demand.
export function createAPIClient({ version, fetchImpl = (...args) => fetch(...args), onUpdateRequired = () => {} }) {
  let callbacks = { getAccessToken: () => null, onSession: () => {}, onExpired: () => {} }
  let refreshPromise = null
  let epoch = 0

  function configureAuth(next) { callbacks = next }

  async function send(url, options = {}, token = null) {
    const headers = new Headers(options.headers)
    headers.set('Accept', 'application/json')
    headers.set('X-App-Version', version)
    headers.set('X-Client', 'web')
    headers.set('X-Requested-With', 'roadmates')
    if (token) headers.set('Authorization', `Bearer ${token}`)
    else headers.delete('Authorization')
    if (options.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
    const response = await fetchImpl(url, { ...options, credentials: 'include', headers })
    if (response.status === 204) return null
    let body
    try { body = await response.json() }
    catch { throw new APIError(response.status, { message: 'The server could not complete the request. Please try again.' }) }
    if (!response.ok) {
      if (response.status === 426 && body.code === 'APP_VERSION_UNSUPPORTED') onUpdateRequired()
      throw new APIError(response.status, body)
    }
    return body
  }

  function expire() { epoch++; callbacks.onExpired() }

  function refreshSession() {
    if (refreshPromise) return refreshPromise
    const started = epoch
    refreshPromise = (async () => {
      try {
        // No caller AbortSignal: one cancelled request must not cancel the
        // shared refresh or leave a rotated cookie without its access token.
        const session = await send('/api/v1/auth/refresh', { method: 'POST' })
        if (started !== epoch) throw new Error('The session changed during refresh.')
        callbacks.onSession(session)
        return session
      } catch (error) {
        if (started === epoch) expire()
        throw error
      } finally { refreshPromise = null }
    })()
    return refreshPromise
  }

  async function request(path, options = {}) {
    const started = epoch
    const originalToken = callbacks.getAccessToken()
    try { return await send(`/api/v1${path}`, options, originalToken) }
    catch (error) {
      if (error.status !== 401) throw error
      // Do not replay an old account's request after logout/login, or start
      // another rotation when a parallel failure already ended the session.
      if (started !== epoch) throw error
      // A parallel request may already have completed the refresh.
      if (!callbacks.getAccessToken() || callbacks.getAccessToken() === originalToken) await refreshSession()
      if (started !== epoch) throw error
      options.signal?.throwIfAborted()
      try { return await send(`/api/v1${path}`, options, callbacks.getAccessToken()) }
      catch (retryError) { if (retryError.status === 401) expire(); throw retryError }
    }
  }

  async function signIn(action, values) {
    epoch++
    const started = epoch
    const session = await send(`/api/v1/auth/${action}`, { method: 'POST', body: JSON.stringify(values) })
    if (started !== epoch) throw new Error('The session changed. Please try again.')
    callbacks.onSession(session)
    return session
  }

  async function logout() {
    epoch++
    // Let an existing rotation finish before the server revokes its family.
    if (refreshPromise) await refreshPromise.catch(() => {})
    await send('/api/v1/auth/logout', { method: 'POST' })
    expire()
  }

  return { configureAuth, request, refreshSession, login: (values) => signIn('login', values), register: (values) => signIn('register', values), logout, version: (options) => send('/api/version', options) }
}
