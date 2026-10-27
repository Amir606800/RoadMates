import { Fragment, useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { listDrives } from '../api/rides.js'
import { useApp } from '../context/AppContext.jsx'
import DriveCard from '../components/DriveCard.jsx'
import SearchForm from '../components/SearchForm.jsx'
import EmptyState from '../components/EmptyState.jsx'
import Icon from '../components/Icon.jsx'
import AdSlot from '../components/AdSlot.jsx'
import { formatDate } from '../utils/format.js'

export default function DrivesPage() {
  const { role, isVerified, search, setSearch } = useApp()
  const [drives, setDrives] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [request, setRequest] = useState(0)
  const hasRoute = Boolean(search.from && search.to)
  useEffect(() => {
    if (!hasRoute || role === 'driver') return
    const controller = new AbortController()
    setLoading(true)
    setError('')
    listDrives(search, { signal: controller.signal }).then((items) => {
      if (!controller.signal.aborted) setDrives(items)
    }).catch((err) => {
      if (err.name !== 'AbortError') setError('We couldn’t load these drives. Please try again.')
    }).finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => controller.abort()
  }, [search, request, hasRoute, role])

  if (role === 'driver' && isVerified) return <Navigate to="/driver/drives" replace />
  const available = drives.filter((drive) => drive.seats_available >= Number(search.seats))
  return <>
    <section className="page-intro home-intro"><div><span className="eyebrow">GOOD COMPANY. GREAT JOURNEYS.</span><h1>A little further.<br /><span>Together.</span></h1><p>Share the road. Make a connection.</p></div><div className="journey-art" aria-hidden="true"><span className="art-sun" /><Icon name="road" size={85} /></div></section>
    <SearchForm search={search} loading={loading} onSearch={(values) => { setSearch(values); setRequest((value) => value + 1) }} />
    <div className="trust-line"><Icon name="shield" size={14} /><span>Verified drivers</span><i /><span>Better journeys</span><Icon name="leaf" size={14} /></div>
    <div className="section-heading"><h2>Available drives</h2><span>{formatDate(search.date, { weekday: 'short' })}</span></div>
    {hasRoute && <div className="row results-caption"><span className="live-dot" /><p>{loading ? 'Looking for your next ride…' : `${available.length} ${available.length === 1 ? 'drive' : 'drives'} from ${search.from.name_az} to ${search.to.name_az}`}</p></div>}
    <div className="stack" aria-busy={loading}>
      {!hasRoute ? <EmptyState title="Where are you headed?">Choose your departure and destination districts to find a shared ride.</EmptyState>
        : loading ? <div className="card" role="status">Loading available drives…</div>
          : error ? <EmptyState title="A bump in the road" action={<button className="button button-secondary" onClick={() => setRequest((n) => n + 1)}>Try again</button>}>{error}</EmptyState>
            : available.length ? available.map((drive, index) => <Fragment key={drive.id}><DriveCard drive={drive} />{(index + 1) % 3 === 0 && <AdSlot placement="drives-list-inline" />}</Fragment>)
              : <EmptyState title="No drives just yet">Try another district, a different date, or fewer seats.</EmptyState>}
    </div>
    <p className="page-footnote"><Icon name="leaf" size={14} /> A shared ride. A smaller footprint.</p>
  </>
}
