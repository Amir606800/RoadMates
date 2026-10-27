import { useEffect, useSyncExternalStore } from 'react'
import { getAPIVersion, getUpdateRequired, subscribeUpdateRequired } from '../api/client.js'

export default function AppCompatibility({ children }) {
  const updateRequired = useSyncExternalStore(subscribeUpdateRequired, getUpdateRequired, getUpdateRequired)
  useEffect(() => {
    const controller = new AbortController()
    // Session restoration handles connection failures; this checks compatibility.
    getAPIVersion({ signal: controller.signal }).catch(() => {})
    return () => controller.abort()
  }, [])
  if (updateRequired) return <main className="app-update" role="alert">
    <span className="eyebrow">ROADMATES UPDATE</span>
    <h1>Please update / reload the app</h1>
    <p>A newer version is required to keep using RoadMates.</p>
    <button className="button button-primary" onClick={() => window.location.reload()}>Reload app</button>
  </main>
  return children
}
