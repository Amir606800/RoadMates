import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listDriverApplications } from '../api/driverApplications.js'
import Icon from '../components/Icon.jsx'
import EmptyState from '../components/EmptyState.jsx'

export default function AdminPage() {
  const [status, setStatus] = useState('pending')
  const [offset, setOffset] = useState(0)
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [retry, setRetry] = useState(0)
  useEffect(() => {
    const controller = new AbortController()
    setLoading(true); setError(''); setResult(null)
    listDriverApplications(status, offset, { signal: controller.signal })
      .then((data) => { if (!controller.signal.aborted) setResult(data) })
      .catch((err) => { if (err.name !== 'AbortError') setError(err.message || 'We could not load applications.') })
      .finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => controller.abort()
  }, [status, offset, retry])
  return <>
    <Link className="back-link" to="/profile"><Icon name="arrowLeft" size={17} />Your profile</Link>
    <section className="page-intro"><span className="eyebrow">ROADMATES ADMIN</span><h1>Welcome new drivers.</h1><p>Review applications and help keep every journey safe.</p></section>
    <div className="admin-filters" role="group" aria-label="Application status">{['pending', 'approved', 'rejected'].map((value) => <button key={value} aria-pressed={status === value} className={status === value ? 'selected' : ''} onClick={() => { setStatus(value); setOffset(0) }}>{value.charAt(0).toUpperCase() + value.slice(1)}</button>)}</div>
    {loading ? <div className="card" role="status">Loading applications…</div> : error ? <div className="card stack"><p className="field-error" role="alert">{error}</p><button className="button button-secondary" onClick={() => setRetry((current) => current + 1)}>Try again</button></div> : <>
      <div className="stack">{result?.applications.length ? result.applications.map((app) => <Link className="card application-card" to={`/admin/driver-applications/${app.id}`} key={app.id}><div className="row between"><h2>{app.first_name} {app.last_name}</h2><span className={`badge ${app.status}`}>{app.status}</span></div><p>{app.car_make} {app.car_model} · {app.car_plate}</p><p>{app.contact_phone}</p><div className="row between"><time dateTime={app.created_at}>{new Date(app.created_at).toLocaleDateString()}</time><span className="row text-sm">View application<Icon name="chevron" size={16} /></span></div></Link>) : <EmptyState icon="shield" title={`No ${status} applications`}>Applications will appear here when they are ready.</EmptyState>}</div>
      {(offset > 0 || result?.pagination.has_more) && <div className="application-pagination"><button className="button button-secondary" disabled={offset === 0} onClick={() => setOffset((current) => Math.max(0, current - 20))}>Previous</button><span>Page {Math.floor(offset / 20) + 1}</span><button className="button button-secondary" disabled={!result?.pagination.has_more} onClick={() => setOffset((current) => current + 20)}>Next</button></div>}
    </>}
  </>
}
