import { useEffect, useId, useRef, useState } from 'react'
import { listDistricts } from '../api/rides.js'
import Icon from './Icon.jsx'

export default function DistrictPicker({ id, label, value, onChange, excludeId, disabled, error }) {
  const dialog = useRef(null)
  const trigger = useRef(null)
  const input = useRef(null)
  const titleId = useId()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [failure, setFailure] = useState('')
  const [retry, setRetry] = useState(0)

  useEffect(() => {
    if (!open) return
    dialog.current.showModal()
    input.current.focus()
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = previous; dialog.current?.close() }
  }, [open])

  useEffect(() => {
    if (!open) return
    const controller = new AbortController()
    setLoading(true)
    setFailure('')
    setItems([])
    const timer = setTimeout(() => {
      listDistricts(query, { signal: controller.signal }).then((result) => {
        if (!controller.signal.aborted) setItems(result)
      }).catch((err) => {
        if (err.name !== 'AbortError') setFailure('Could not load districts. Please try again.')
      }).finally(() => { if (!controller.signal.aborted) setLoading(false) })
    }, query ? 200 : 0)
    return () => { clearTimeout(timer); controller.abort() }
  }, [open, query, retry])

  function close() { setOpen(false); trigger.current?.focus() }
  return <div className="field district-field">
    <label htmlFor={id}>{label}</label>
    <button ref={trigger} type="button" id={id} className="district-trigger" aria-haspopup="dialog" aria-invalid={Boolean(error)} aria-describedby={error ? `${id}-error` : value?.requires_permit ? `${id}-hint` : undefined} disabled={disabled} onClick={() => { setQuery(''); setOpen(true) }}>
      <span className={value ? '' : 'muted'}>{value?.name_az || 'Choose district'}</span><Icon name="search" size={16} />
    </button>
    {value?.requires_permit && <span className="field-hint permit-hint" id={`${id}-hint`}>Permit may be required</span>}
    {error && <span className="field-error" id={`${id}-error`}>{error}</span>}
    {open && <dialog ref={dialog} className="district-dialog" aria-labelledby={titleId} onCancel={(event) => { event.preventDefault(); close() }} onClick={(event) => { if (event.target === dialog.current) close() }}>
      <div className="district-sheet">
        <div className="row between"><h2 id={titleId}>{label}: choose a district</h2><button type="button" className="icon-button" aria-label="Close district picker" onClick={close}><Icon name="close" size={20} /></button></div>
        <label className="sr-only" htmlFor={`${id}-query`}>Search districts</label>
        <input ref={input} id={`${id}-query`} type="search" placeholder="Search districts…" value={query} maxLength={80} autoComplete="off" onChange={(event) => setQuery(event.target.value)} />
        <p className="field-hint">You can search without Azerbaijani letters.</p>
        <div className="district-options" aria-busy={loading}>
          {loading ? <p role="status" className="muted">Loading districts…</p>
            : failure ? <div role="alert"><p>{failure}</p><button type="button" className="button button-secondary" onClick={() => setRetry((n) => n + 1)}>Try again</button></div>
              : items.length ? items.map((district) => <button type="button" key={district.id} className="district-option" aria-pressed={value?.id === district.id} disabled={excludeId === district.id} onClick={() => { onChange(district); close() }}>
                <span><strong>{district.name_az}</strong>{district.requires_permit && <small>Permit may be required</small>}{excludeId === district.id && <small>Already selected for the other stop</small>}</span>
                {value?.id === district.id && <Icon name="check" size={18} />}
              </button>) : <p role="status" className="muted">No districts match your search.</p>}
        </div>
      </div>
    </dialog>}
  </div>
}
