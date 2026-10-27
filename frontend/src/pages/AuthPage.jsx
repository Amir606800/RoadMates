import { useState } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext.jsx'
import { SessionLoading } from '../components/RequireAuth.jsx'
import Field from '../components/Field.jsx'
import Icon from '../components/Icon.jsx'

const initial = { email: '', password: '', first_name: '', last_name: '', phone: '' }
const passwordHint = 'At least 8 characters, at most 72 UTF-8 bytes. Include uppercase, lowercase, and a special character. No spaces.'

export default function AuthPage({ register = false }) {
  const auth = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [values, setValues] = useState(initial)
  const [errors, setErrors] = useState({})
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState('')
  const destination = typeof location.state?.from === 'string' && location.state.from.startsWith('/') && !location.state.from.startsWith('//') ? location.state.from : '/drives'
  if (auth.status === 'loading') return <SessionLoading />
  if (auth.user) return <Navigate to={destination} replace />

  function change(event) { setValues((current) => ({ ...current, [event.target.name]: event.target.value })); setErrors((current) => ({ ...current, [event.target.name]: '' })); setMessage('') }
  async function submit(event) {
    event.preventDefault()
    if (submitting) return
    const nextErrors = {}
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email.trim()) || values.email.trim().length > 254) nextErrors.email = 'Enter a valid email address.'
    if (!values.password) nextErrors.password = 'Enter your password.'
    if (register) {
      if ([...values.password].length < 8 || new TextEncoder().encode(values.password).length > 72 || !/\p{Lu}/u.test(values.password) || !/\p{Ll}/u.test(values.password) || !/[\p{P}\p{S}]/u.test(values.password) || /[\s\p{Cc}]/u.test(values.password)) nextErrors.password = passwordHint
      for (const key of ['first_name', 'last_name']) if (!values[key].trim() || [...values[key].trim()].length > 60 || !/^[\p{L}\p{M} '\-’]+$/u.test(values[key].trim())) nextErrors[key] = 'Enter a name of up to 60 characters.'
      const digits = values.phone.replace(/[^0-9]/g, '')
      if (digits.length < 7 || digits.length > 15 || values.phone.trim().length > 25 || !/^\+?[0-9 ()-]+$/.test(values.phone.trim())) nextErrors.phone = 'Enter a valid contact phone.'
    }
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length) { document.getElementById(`auth-${Object.keys(nextErrors)[0]}`)?.focus(); return }
    setSubmitting(true); setMessage('')
    try {
      const body = { email: values.email.trim().toLowerCase(), password: values.password }
      if (register) Object.assign(body, { first_name: values.first_name.trim(), last_name: values.last_name.trim(), phone: values.phone.trim() })
      await (register ? auth.register(body) : auth.login(body))
      setValues(initial)
      navigate(destination, { replace: true })
    } catch (error) { setMessage(error.message || 'We could not connect. Please try again.') }
    finally { setSubmitting(false) }
  }
  function field(name, label, props = {}) {
    return <Field id={`auth-${name}`} name={name} label={label} value={values[name]} onChange={change} error={errors[name]} disabled={submitting} required {...props} />
  }
  return <main className="auth-shell">
    <Link className="brand auth-brand" to="/login"><span className="brand-mark"><Icon name="road" size={21} /></span>RoadMates</Link>
    <section className="page-intro"><span className="eyebrow">GOOD PEOPLE. SHARED JOURNEYS.</span><h1>{register ? 'Find your people.\nShare the road.' : 'Welcome back.\nLet’s hit the road.'}</h1><p>{register ? 'Create your account for the journey ahead.' : 'Sign in to pick up where you left off.'}</p></section>
    <form className="card stack" onSubmit={submit} noValidate>
      {register && <div className="form-grid">{field('first_name', 'First name', { autoComplete: 'given-name', maxLength: 60 })}{field('last_name', 'Last name', { autoComplete: 'family-name', maxLength: 60 })}</div>}
      {field('email', 'Email', { type: 'email', autoComplete: 'email', autoCapitalize: 'none', spellCheck: false, maxLength: 254 })}
      {register && field('phone', 'Phone', { type: 'tel', autoComplete: 'tel', maxLength: 25, placeholder: '+994 50 000 00 00' })}
      {field('password', 'Password', { type: 'password', autoComplete: register ? 'new-password' : 'current-password', maxLength: 72, hint: register ? passwordHint : undefined })}
      {(message || (auth.status === 'error' && auth.restoreError)) && <p className="field-error" role="alert">{message || auth.restoreError}</p>}
      <button className="button full-width" disabled={submitting}>{submitting ? 'Please wait…' : register ? 'Create account' : 'Sign in'}<Icon name="arrow" size={18} /></button>
    </form>
    <p className="auth-switch">{register ? 'Already a road mate?' : 'New to RoadMates?'} <Link to={register ? '/login' : '/register'} state={location.state}>{register ? 'Sign in' : 'Create an account'}</Link></p>
  </main>
}
