import { useState } from 'react'
import { dateAfter } from '../utils/format.js'
import DistrictPicker from './DistrictPicker.jsx'
import Icon from './Icon.jsx'

export default function SearchForm({ search, onSearch, loading }) {
  const [values, setValues] = useState(search)
  const [error, setError] = useState('')
  function update(key, value) { setValues((current) => ({ ...current, [key]: value })); setError('') }
  function submit(event) {
    event.preventDefault()
    if (!values.from || !values.to) { setError('Choose both districts.'); return }
    if (values.from.id === values.to.id) { setError('Choose a different destination.'); return }
    setError('')
    onSearch(values)
  }
  return <form className="card search-card" onSubmit={submit}>
    <div className="search-cities">
      <div className="search-city"><span className="origin-dot" /><DistrictPicker id="search-from" label="From" value={values.from} excludeId={values.to?.id} onChange={(value) => update('from', value)} /></div>
      <button className="swap-button icon-button" type="button" aria-label="Swap departure and destination" onClick={() => { setValues({ ...values, from: values.to, to: values.from }); setError('') }}><Icon name="swap" size={19} /></button>
      <div className="search-city"><Icon name="pin" size={17} /><DistrictPicker id="search-to" label="To" value={values.to} excludeId={values.from?.id} onChange={(value) => update('to', value)} /></div>
    </div>
    <div className="search-options"><div className="field"><label htmlFor="search-date"><Icon name="calendar" size={14} /> Departure (AZT)</label><input id="search-date" type="date" name="date" min={dateAfter()} value={values.date} onChange={(event) => update('date', event.target.value)} required /></div><div className="field"><label htmlFor="search-seats"><Icon name="users" size={14} /> Seats</label><select id="search-seats" name="seats" value={values.seats} onChange={(event) => update('seats', event.target.value)}>{[1, 2, 3, 4, 5, 6, 7, 8].map((n) => <option value={n} key={n}>{n} {n === 1 ? 'seat' : 'seats'}</option>)}</select></div></div>
    {error && <p className="field-error" role="alert">{error}</p>}
    <button className="button full-width" disabled={loading}><Icon name="search" size={18} />{loading ? 'Finding your drive…' : 'Find a drive'}<Icon name="arrow" size={18} /></button>
  </form>
}
