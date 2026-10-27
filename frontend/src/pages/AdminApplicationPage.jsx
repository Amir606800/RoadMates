import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getAdminApplication, reviewDriverApplication } from '../api/driverApplications.js'
import Icon from '../components/Icon.jsx'
import Field from '../components/Field.jsx'

function timestamp(value) { return new Date(value).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' }) }

export default function AdminApplicationPage() {
  const { applicationId } = useParams()
  const [application, setApplication] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [retry, setRetry] = useState(0)
  const [reason, setReason] = useState('')
  const [showPIN, setShowPIN] = useState(false)
  const [reviewing, setReviewing] = useState('')
  const [notice, setNotice] = useState('')
  const generation = useRef(0)
  useEffect(() => {
    generation.current++
    const controller = new AbortController()
    setLoading(true); setError(''); setApplication(null); setShowPIN(false); setReason(''); setNotice(''); setReviewing('')
    getAdminApplication(applicationId, { signal: controller.signal })
      .then((detail) => { if (!controller.signal.aborted) setApplication(detail) })
      .catch((err) => { if (err.name !== 'AbortError') setError(err.message || 'We could not load this application.') })
      .finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => { generation.current++; controller.abort() }
  }, [applicationId, retry])

  async function review(action) {
    if (reviewing) return
    const started = generation.current
    setReviewing(action); setError(''); setNotice('')
    try {
      const result = await reviewDriverApplication(applicationId, action, action === 'reject' ? reason : '')
      if (started !== generation.current) return
      setApplication((current) => ({ ...current, ...result }))
      setShowPIN(false); setReason('')
      setNotice(action === 'approve' ? 'Application approved. Driver access is now available on the applicant’s next session refresh.' : 'Application rejected. The applicant can correct their details and apply again.')
    } catch (err) {
      if (started !== generation.current) return
      setError(err.message || 'We could not save this decision. Please try again.')
      if (err.code === 'APPLICATION_ALREADY_REVIEWED') {
        try { const latest = await getAdminApplication(applicationId); if (started === generation.current) { setApplication(latest); setShowPIN(false) } } catch {}
      }
    } finally { if (started === generation.current) setReviewing('') }
  }

  return <>
    <Link className="back-link" to="/admin"><Icon name="arrowLeft" size={17} />Driver applications</Link>
    <section className="page-intro"><span className="eyebrow">DRIVER VERIFICATION</span><h1>Review the details.</h1><p>Check the applicant’s identity and vehicle before making a decision.</p></section>
    {loading ? <div className="card" role="status">Loading application…</div> : application ? <div className="stack">
      <section className="card stack"><div className="row between"><h2>{application.first_name} {application.last_name}</h2><span className={`badge ${application.status}`}>{application.status}</span></div>
        <dl className="application-details"><div><dt>Contact phone</dt><dd>{application.contact_phone}</dd></div><div><dt>Submitted</dt><dd>{timestamp(application.created_at)}</dd></div><div><dt>Personal PIN</dt><dd className="pin-review"><span className="personal-pin">{showPIN ? application.pin_code : '•••••••'}</span><button className="button button-secondary" onClick={() => setShowPIN((current) => !current)} aria-pressed={showPIN}>{showPIN ? 'Hide PIN' : 'Show PIN'}</button></dd></div></dl>
      </section>
      <section className="card stack"><h2>Vehicle</h2><dl className="application-details"><div><dt>Make and model</dt><dd>{application.car_make} {application.car_model}</dd></div><div><dt>Year and color</dt><dd>{application.car_year} · {application.car_color}</dd></div><div><dt>Plate number</dt><dd>{application.car_plate}</dd></div><div><dt>Passenger seats</dt><dd>{application.seats}</dd></div></dl></section>
      {application.status === 'pending' ? <section className="card stack"><h2>Your decision</h2><Field id="reject-reason" label="Reason for rejection (optional)" hint="Visible to the applicant. Up to 500 characters."><textarea id="reject-reason" value={reason} onChange={(event) => setReason(event.target.value)} maxLength={500} disabled={Boolean(reviewing)} rows={3} /></Field><div className="review-actions"><button className="button" disabled={Boolean(reviewing)} onClick={() => review('approve')}>{reviewing === 'approve' ? 'Approving…' : 'Approve driver'}</button><button className="button button-danger" disabled={Boolean(reviewing)} onClick={() => review('reject')}>{reviewing === 'reject' ? 'Rejecting…' : 'Reject application'}</button></div></section> : <section className="card stack"><h2>Review complete</h2>{application.reviewed_at && <p className="text-sm muted">Reviewed {timestamp(application.reviewed_at)}</p>}{application.reject_reason && <p className="review-reason">{application.reject_reason}</p>}</section>}
      {notice && <p className="notice" role="status">{notice}</p>}
      {error && <p className="field-error" role="alert">{error}</p>}
    </div> : <div className="card stack"><p className="field-error" role="alert">{error || 'Application not found.'}</p><button className="button button-secondary" onClick={() => setRetry((current) => current + 1)}>Try again</button></div>}
  </>
}
