import { useState } from 'react'
import { Link } from 'react-router-dom'
import { applyToDrive } from '../api/driverApplications.js'
import { buildDriverApplication } from '../api/driverApplicationPayload.js'
import { useApp } from '../context/AppContext.jsx'
import { validateVerification } from '../utils/validation.js'
import Field from '../components/Field.jsx'
import Icon from '../components/Icon.jsx'
import VerificationStatus from '../components/VerificationStatus.jsx'

function ApplicationForm() {
  const { user, application, acceptApplication, reloadApplication } = useApp()
  const [values, setValues] = useState(() => ({ phone: application?.contact_phone || user.phone, name: application?.first_name || user.name, surname: application?.last_name || user.surname, pin: '', ...user.car }))
  const [errors, setErrors] = useState({})
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  function change(event) { setValues({ ...values, [event.target.name]: event.target.value }); setErrors({ ...errors, [event.target.name]: '' }) }
  async function submit(event) {
    event.preventDefault()
    if (submitting) return
    const nextErrors = validateVerification(values)
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length) { document.getElementById(`verify-${Object.keys(nextErrors)[0]}`)?.focus(); return }
    const payload = buildDriverApplication(values)
    setValues((current) => ({ ...current, pin: '' }))
    setSubmitting(true); setError('')
    try { acceptApplication(await applyToDrive(payload)) }
    catch (err) {
      setError(err.message || 'We could not submit your application. Please re-enter your PIN and try again.')
      if (err.status === 409) await reloadApplication().catch(() => {})
    } finally { setSubmitting(false) }
  }
  function field(name, label, props = {}) {
    return <Field id={`verify-${name}`} name={name} label={label} value={values[name]} error={errors[name]} onChange={change} disabled={submitting} required {...props} />
  }
  return <form className="stack verification-form" noValidate onSubmit={submit}>
    <section className="card stack"><div className="form-section-title"><span>01</span><div><h2>About you</h2><p>All fields are required.</p></div></div>
      {field('phone', 'Contact phone', { type: 'tel', autoComplete: 'tel', placeholder: '+994 50 000 00 00', maxLength: 25 })}
      <div className="form-grid">{field('name', 'Name', { autoComplete: 'given-name', maxLength: 60 })}{field('surname', 'Surname', { autoComplete: 'family-name', maxLength: 60 })}</div>
      {field('pin', 'Personal PIN code', { type: 'password', autoComplete: 'off', autoCapitalize: 'characters', spellCheck: false, maxLength: 7, placeholder: '7 letters and numbers', hint: 'Encrypted securely and visible only to administrators reviewing your application.' })}
    </section>
    <section className="card stack"><div className="form-section-title"><span>02</span><div><h2>Your car</h2><p>Make your future passengers feel at home.</p></div></div>
      <div className="form-grid">{field('brand', 'Brand', { placeholder: 'Toyota', maxLength: 50 })}{field('model', 'Model', { placeholder: 'Corolla', maxLength: 50 })}</div>
      <div className="form-grid">{field('year', 'Year', { type: 'number', min: 1980, max: new Date().getFullYear() + 1, inputMode: 'numeric' })}{field('color', 'Color', { placeholder: 'Silver', maxLength: 50 })}</div>
      {field('plate', 'Plate number', { placeholder: '10-RM-245', maxLength: 15, autoCapitalize: 'characters' })}
      {field('seats', 'Passenger seats', { type: 'number', min: 1, max: 8, inputMode: 'numeric', hint: 'Number of seats available, excluding the driver.' })}
    </section>
    {error && <p className="field-error" role="alert">{error}</p>}
    <button className="button full-width" disabled={submitting}>{submitting ? 'Submitting application…' : 'Submit for verification'}<Icon name="arrow" size={18} /></button>
  </form>
}

export default function VerificationPage() {
  const { user, application, applicationStatus, applicationError, reloadApplication, isVerified, setRole } = useApp()
  const [checking, setChecking] = useState(false)
  async function check() { setChecking(true); try { await reloadApplication() } catch {} finally { setChecking(false) } }
  const canApply = !isVerified && (!application || application.status === 'rejected')
  return <>
    <Link to="/profile" className="back-link"><Icon name="arrowLeft" size={17} />Your profile</Link>
    <section className="page-intro"><span className="eyebrow">TAKE THE WHEEL</span><h1>Good drives.<br />Start with trust.</h1><p>A few details help us keep every journey safe.</p></section>
    {applicationStatus === 'loading' ? <div className="card" role="status">Checking your application…</div> : <>
      {applicationStatus === 'error' ? <div className="card stack"><p className="field-error" role="alert">{applicationError}</p><button className="button button-secondary" disabled={checking} onClick={check}>Try again</button></div> : <>
        <VerificationStatus verification={user.verification} />
        {canApply ? <ApplicationForm key={application?.id || user.id} /> : <div className="stack verification-form">
          {application && <section className="card stack"><h2>Your application</h2><p>{application.first_name} {application.last_name}</p><p className="text-sm muted">{application.contact_phone}</p><p>{application.car_color} {application.car_make} {application.car_model} · {application.car_year}</p><p className="text-sm muted">{application.car_plate} · {application.seats} passenger seats</p></section>}
          {isVerified ? <Link className="button full-width" to="/driver/drives" onClick={() => setRole('driver')}>Manage your drives<Icon name="arrow" size={18} /></Link> : <><p className="field-hint">Your status updates automatically. Driver access unlocks after approval.</p><button className="button button-secondary full-width" onClick={check} disabled={checking}>{checking ? 'Checking…' : 'Check application status'}</button></>}
        </div>}
      </>}
    </>}
  </>
}
