export default function Field({ label, id, error, hint, children, ...inputProps }) {
  const description = [error && `${id}-error`, hint && `${id}-hint`].filter(Boolean).join(' ') || undefined
  return <div className="field">
    <label htmlFor={id}>{label}</label>
    {children || <input id={id} aria-invalid={Boolean(error)} aria-describedby={description} {...inputProps} />}
    {hint && <span className="field-hint" id={`${id}-hint`}>{hint}</span>}
    {error && <span className="field-error" id={`${id}-error`}>{error}</span>}
  </div>
}
