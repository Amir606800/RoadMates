import assert from 'node:assert/strict'
import test from 'node:test'
import { createAPIClient } from '../src/api/httpClient.js'

function response(status, body) { return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }) }
function setup(fetchImpl) {
  let token = 'old-token', user = null, cleared = 0
  const client = createAPIClient({ version: '0.1.0', fetchImpl })
  client.configureAuth({ getAccessToken: () => token, onSession: (session) => { token = session.access_token; user = session.user }, onExpired: () => { token = null; user = null; cleared++ } })
  return { client, state: () => ({ token, user, cleared }) }
}

test('parallel 401s share one rotation and retry with the new token', async () => {
  let refreshes = 0, retries = 0
  const { client, state } = setup(async (url, options) => {
    assert.equal(options.credentials, 'include')
    assert.equal(options.headers.get('X-App-Version'), '0.1.0')
    assert.equal(options.headers.get('X-Client'), 'web')
    assert.equal(options.headers.get('X-Requested-With'), 'roadmates')
    if (url.endsWith('/refresh')) { refreshes++; await new Promise((resolve) => setTimeout(resolve, 10)); return response(200, { access_token: 'new-token', user: { id: 'u' } }) }
    if (options.headers.get('Authorization') === 'Bearer old-token') return response(401, { code: 'UNAUTHORIZED' })
    assert.equal(options.headers.get('Authorization'), 'Bearer new-token'); retries++
    return response(200, { id: 'u' })
  })
  const results = await Promise.all(Array.from({ length: 8 }, () => client.request('/auth/me')))
  assert.equal(results.length, 8); assert.equal(refreshes, 1); assert.equal(retries, 8)
  assert.equal(state().token, 'new-token')
})

test('a failed shared refresh clears authentication and does not retry requests', async () => {
  let refreshes = 0
  const { client, state } = setup(async (url) => {
    if (url.endsWith('/refresh')) { refreshes++; await new Promise((resolve) => setTimeout(resolve, 5)) }
    return response(401, { code: 'UNAUTHORIZED' })
  })
  const results = await Promise.allSettled([client.request('/auth/me'), client.request('/auth/me')])
  assert.ok(results.every((result) => result.status === 'rejected'))
  assert.equal(refreshes, 1); assert.equal(state().token, null); assert.equal(state().cleared, 1)
})

test('a retry that remains unauthorized does not rotate a second time', async () => {
  let refreshes = 0, requests = 0
  const { client, state } = setup(async (url) => {
    if (url.endsWith('/refresh')) { refreshes++; return response(200, { access_token: 'new', user: { id: 'u' } }) }
    requests++; return response(401, { code: 'UNAUTHORIZED' })
  })
  await assert.rejects(client.request('/auth/me'))
  assert.equal(refreshes, 1); assert.equal(requests, 2); assert.equal(state().token, null)
})

test('late unauthorized responses do not rotate again after a shared refresh failed', async () => {
  let releaseLate, refreshes = 0
  const late = new Promise((resolve) => { releaseLate = resolve })
  const { client } = setup(async (url) => {
    if (url.endsWith('/refresh')) { refreshes++; return response(401, {}) }
    if (url.endsWith('/late')) await late
    return response(401, {})
  })
  const delayed = client.request('/late').catch(() => {})
  await assert.rejects(client.request('/auth/me'))
  releaseLate(); await delayed
  assert.equal(refreshes, 1)
})

test('login failures never trigger refresh or include an old bearer token', async () => {
  let calls = 0
  const { client } = setup(async (url, options) => {
    calls++; assert.ok(url.endsWith('/login')); assert.equal(options.headers.has('Authorization'), false)
    return response(401, { code: 'INVALID_CREDENTIALS' })
  })
  await assert.rejects(client.login({ email: 'a@example.com', password: 'not-a-real-password' }), { code: 'INVALID_CREDENTIALS' })
  assert.equal(calls, 1)
})

test('logout waits for a rotation and cannot restore authentication afterward', async () => {
  let finishRefresh
  const barrier = new Promise((resolve) => { finishRefresh = resolve })
  const calls = []
  const { client, state } = setup(async (url) => {
    calls.push(url)
    if (url.endsWith('/refresh')) { await barrier; return response(200, { access_token: 'new', user: { id: 'u' } }) }
    return new Response(null, { status: 204 })
  })
  const refresh = client.refreshSession().catch(() => {})
  const logout = client.logout()
  finishRefresh(); await Promise.all([refresh, logout])
  assert.deepEqual(calls, ['/api/v1/auth/refresh', '/api/v1/auth/logout'])
  assert.equal(state().token, null); assert.equal(state().user, null)
})

test('cancelling one request does not abort the shared refresh for other requests', async () => {
  const controller = new AbortController()
  const { client } = setup(async (url, options) => {
    if (url.endsWith('/refresh')) { assert.equal(options.signal, undefined); controller.abort(); await new Promise((resolve) => setTimeout(resolve, 5)); return response(200, { access_token: 'new', user: { id: 'u' } }) }
    if (options.headers.get('Authorization') === 'Bearer old-token') return response(401, {})
    return response(200, { ok: true })
  })
  const results = await Promise.allSettled([client.request('/auth/me', { signal: controller.signal }), client.request('/auth/me')])
  assert.equal(results[0].status, 'rejected'); assert.equal(results[1].status, 'fulfilled')
})
