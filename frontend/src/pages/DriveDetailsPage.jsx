import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getDrive } from '../api/rides.js'
import DriveCard from '../components/DriveCard.jsx'
import Icon from '../components/Icon.jsx'
import EmptyState from '../components/EmptyState.jsx'

export default function DriveDetailsPage() {
  const { driveId } = useParams()
  const [drive, setDrive] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    setDrive(null)
    setError('')
    getDrive(driveId, { signal: controller.signal }).then((item) => {
      if (!controller.signal.aborted) setDrive(item)
    }).catch((err) => { if (err.name !== 'AbortError') setError('This drive could not be loaded. Please try again from the list.') })
      .finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => controller.abort()
  }, [driveId])
  return <>
    <Link to="/drives" className="back-link"><Icon name="arrowLeft" size={17} />All drives</Link>
    {loading ? <div className="card" role="status">Loading drive details…</div>
      : !drive ? <EmptyState title="Drive unavailable">{error}</EmptyState>
        : <>
          <section className="page-intro"><span className="eyebrow">YOUR NEXT JOURNEY</span><h1>{drive.from_district.name_az} <span className="muted">→</span> {drive.to_district.name_az}</h1><p>Status: {drive.status.replaceAll('_', ' ')}</p></section>
          <DriveCard drive={drive} />
          <section className="card stack ride-notes"><h2>Journey notes</h2><p>{drive.notes || 'The driver has not added any notes.'}</p><Link className="button button-secondary" to="/map" state={{ from: drive.from_district.name_az, to: drive.to_district.name_az }}><Icon name="map" size={17} />View route</Link></section>
          <p className="page-footnote">Seat reservations are not available yet.</p>
        </>}
  </>
}
