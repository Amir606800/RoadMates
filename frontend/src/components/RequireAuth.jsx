import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'

export function SessionLoading() {
  return <main className="auth-shell auth-loading" role="status"><span className="eyebrow">ROADMATES</span><h1>Your next journey awaits.</h1><p>Restoring your session…</p></main>
}

export default function RequireAuth({ roles = [], children }) {
  const { user, status, restoreError, retryRestore } = useAuth()
  const location = useLocation()
  if (status === 'loading') return <SessionLoading />
  if (status === 'error') return <main className="auth-shell auth-loading"><h1>A small detour.</h1><p role="alert">{restoreError}</p><button className="button" onClick={retryRestore}>Try again</button></main>
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  if (roles.length && !roles.some((role) => user.roles.includes(role))) return <Navigate to="/profile" replace />
  return children
}
