import { useCallback, useEffect, useRef, useState } from 'react'
import { createDrive, finishDrive, getActiveDrive } from '../api/rides.js'
import { useApp } from '../context/AppContext.jsx'
import { dateAfter } from '../utils/format.js'
import { validateDrive } from '../utils/validation.js'
import Field from '../components/Field.jsx'
import Icon from '../components/Icon.jsx'
import DistrictPicker from '../components/DistrictPicker.jsx'
import DriveCard from '../components/DriveCard.jsx'

const initialValues = () => ({ from: null, to: null, date: dateAfter(1), time: '09:00', seats: '1', price: '20', note: '' })

export default function DriverDrivesPage() {
  const { user } = useApp()
  const [values, setValues] = useState(initialValues)
  const [errors, setErrors] = useState({})
  const [active, setActive] = useState(null)
  const [status, setStatus] = useState('loading')
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const mounted = useRef(false)
  const generation = useRef(0)
  const busy = useRef(false)

  const reload = useCallback(async (options = {}) => {
    const request = ++generation.current
    setStatus('loading')
    try {
      const ride = await getActiveDrive(options)
      if (mounted.current && !options.signal?.aborted && request === generation.current) { setActive(ride); setStatus('ready') }
    } catch (err) {
      if (err.name !== 'AbortError' && mounted.current && request === generation.current) {
        setStatus('error'); setError('We could not check your active drive. Please try again.')
      }
    }
  }, [])
  useEffect(() => {
    mounted.current = true
    const controller = new AbortController()
    reload({ signal: controller.signal })
    const focus = () => { if (!busy.current) { setError(''); reload() } }
    window.addEventListener('focus', focus)
    return () => { mounted.current = false; controller.abort(); window.removeEventListener('focus', focus) }
  }, [reload])

  function update(name, value) { setValues((current) => ({ ...current, [name]: value })); setErrors((current) => ({ ...current, [name]: '' })) }
  function change(event) { update(event.target.name, event.target.value) }
  async function submit(event) {
    event.preventDefault()
    if (busy.current || status !== 'ready' || active) return
    const nextErrors = validateDrive(values)
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length) { document.getElementById(`create-${Object.keys(nextErrors)[0]}`)?.focus(); return }
    busy.current = true
    generation.current++
    setSubmitting(true); setMessage(''); setError('')
    try {
      const ride = await createDrive(values)
      if (mounted.current) { setActive(ride); setStatus('ready'); setMessage('Your drive is published. Happy travels!'); setValues(initialValues()) }
    } catch (err) {
      if (!mounted.current) return
      if (err.code === 'DRIVER_HAS_ACTIVE_RIDE') { setMessage('You already have an active drive. Complete or cancel it before publishing another.'); await reload() }
      else { setError(err.status === 400 ? 'Check your districts, future departure, seats, and price.' : 'Your drive could not be published. Check your active drive before trying again.'); await reload() }
    } finally { busy.current = false; if (mounted.current) setSubmitting(false) }
  }
  async function finish(action) {
    if (busy.current || !active) return
    busy.current = true; generation.current++; setSubmitting(true); setMessage(''); setError('')
    try {
      await finishDrive(active.id, action)
      if (mounted.current) { setMessage(action === 'cancel' ? 'Drive cancelled. You can publish a new drive.' : 'Drive completed. You can publish a new drive.'); await reload() }
    } catch (err) {
      if (mounted.current) { setError(err.code === 'RIDE_NOT_ACTIVE' ? 'This drive has already ended. Your active drive has been refreshed.' : 'We could not update this drive. Please check its status and try again.'); await reload() }
    } finally { busy.current = false; if (mounted.current) setSubmitting(false) }
  }
  function field(name, label, props = {}) { return <Field id={`create-${name}`} name={name} label={label} value={values[name]} error={errors[name]} onChange={change} disabled={submitting} {...props} /> }
  return <>
    <section className="page-intro"><span className="eyebrow">A SEAT FOR SOMEONE NEW</span><h1>Your road.<br />Their next adventure.</h1><p>Share your journey, split the cost.</p></section>
    {message && <div className="notice settings-message" role="status"><Icon name="check" /><p>{message}</p></div>}
    {error && <p className="field-error ride-error" role="alert">{error}</p>}
    {status === 'loading' ? <div className="card" role="status">Checking your active drive…</div>
      : status === 'error' ? <div className="card stack"><p>Check your active drive before publishing.</p><button className="button button-secondary" onClick={() => { setError(''); reload() }}>Try again</button></div>
        : active ? <section className="stack active-drive" aria-label="Your active drive"><div className="row between"><h2>Your active drive</h2><span className="badge">{active.status.replaceAll('_', ' ')}</span></div><DriveCard drive={active} /><p className="field-hint">Complete or cancel this drive to publish another.</p><div className="form-grid"><button className="button button-secondary" disabled={submitting} onClick={() => finish('cancel')}>Cancel drive</button><button className="button" disabled={submitting} onClick={() => finish('complete')}>Complete drive</button></div></section>
          : <form className="card stack" noValidate onSubmit={submit}><div className="row between"><h2>Create a drive</h2><span className="badge"><Icon name="shield" size={12} />Verified driver</span></div>
            <div className="form-grid"><DistrictPicker id="create-from" label="From" value={values.from} excludeId={values.to?.id} onChange={(value) => update('from', value)} error={errors.from} disabled={submitting} /><DistrictPicker id="create-to" label="To" value={values.to} excludeId={values.from?.id} onChange={(value) => update('to', value)} error={errors.to} disabled={submitting} /></div>
            <div className="form-grid">{field('date', 'Date (Azerbaijan)', { type: 'date', min: dateAfter() })}{field('time', 'Departure (AZT)', { type: 'time' })}</div>
            <div className="form-grid">{field('seats', 'Available seats', { type: 'number', min: 1, max: 8, inputMode: 'numeric' })}{field('price', 'Price per seat (AZN)', { type: 'number', min: 0, max: 999999.99, step: .01, inputMode: 'decimal' })}</div>
            <Field id="create-note" label="A note for your mates (optional)" error={errors.note}><textarea id="create-note" name="note" value={values.note} onChange={change} maxLength={300} disabled={submitting} placeholder="Meeting point, luggage, or a coffee stop…" aria-invalid={Boolean(errors.note)} aria-describedby={errors.note ? 'create-note-error' : undefined} /></Field>
            <p className="field-hint"><Icon name="car" size={14} /> {user.car.color} {user.car.brand} {user.car.model}</p>
            <button className="button full-width" disabled={submitting}><Icon name="plus" size={18} />{submitting ? 'Publishing…' : 'Publish drive'}</button>
          </form>}
  </>
}
