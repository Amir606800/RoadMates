import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import { authClient } from '../api/client.js'

const AuthContext = createContext(null)

function expiration(token) {
  try { return JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/'))).exp * 1000 }
  catch { return 0 }
}

export function AuthProvider({ children }) {
  const [accessToken, setAccessToken] = useState(null)
  const tokenRef = useRef(null)
  const [user, setUser] = useState(null)
  const [status, setStatus] = useState('loading')
  const [restoreError, setRestoreError] = useState('')
  const [restoreAttempt, setRestoreAttempt] = useState(0)

  const acceptSession = useCallback((session) => {
    tokenRef.current = session.access_token
    setAccessToken(session.access_token)
    setUser(session.user)
    setStatus('authenticated')
    setRestoreError('')
  }, [])

  const clearSession = useCallback(() => {
    tokenRef.current = null
    setAccessToken(null)
    setUser(null)
    setStatus('anonymous')
  }, [])

  useEffect(() => {
    let active = true
    authClient.configureAuth({ getAccessToken: () => tokenRef.current, onSession: acceptSession, onExpired: clearSession })
    setStatus('loading')
    authClient.refreshSession().catch((error) => {
      if (!active) return
      if (error.status === 401) setStatus('anonymous')
      else { setStatus('error'); setRestoreError('We could not restore your session. Check your connection and try again.') }
    })
    return () => { active = false }
  }, [acceptSession, clearSession, restoreAttempt])

  useEffect(() => {
    if (!accessToken) return
    const remaining = expiration(accessToken) - Date.now()
    const delay = Math.max(500, remaining > 60000 ? remaining - 30000 : remaining * 0.75)
    const refresh = () => authClient.refreshSession().catch(() => {})
    const timer = setTimeout(refresh, Math.min(delay, 2147483647))
    const onFocus = () => { if (expiration(tokenRef.current || '') - Date.now() < 30000) refresh() }
    window.addEventListener('focus', onFocus)
    return () => { clearTimeout(timer); window.removeEventListener('focus', onFocus) }
  }, [accessToken])

  return <AuthContext.Provider value={{ user, status, restoreError, retryRestore: () => setRestoreAttempt((count) => count + 1), login: authClient.login, register: authClient.register, logout: authClient.logout, refresh: authClient.refreshSession }}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used inside AuthProvider')
  return value
}
