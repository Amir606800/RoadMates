const adsEnabled = import.meta.env.VITE_ADS_ENABLED === 'true'

export default function AdSlot({ placement, fallback }) {
  if (!adsEnabled) return null
  // Future integration: request a creative from an internal ad API using
  // `placement`, or mount an approved provider here. Keep its loading/error
  // fallback inside this component and respect consent before provider calls.
  return fallback ?? <aside className="ad-slot" aria-label="Advertisement" data-placement={placement}><span>ADVERTISEMENT</span><p>A space for something along the way.</p></aside>
}
