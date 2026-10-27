import { useId } from 'react'
import Icon from './Icon.jsx'

export default function StarRating({ value = 0, onChange, label = 'Rating', disabled = false }) {
  const id = useId()
  if (onChange) return <fieldset className="star-input" disabled={disabled}><legend className="sr-only">{label}</legend>{[1, 2, 3, 4, 5].map((star) => <label className="star-option" key={star}><input type="radio" name={id} value={star} checked={value === star} onChange={() => onChange(star)} aria-label={`${star} ${star === 1 ? 'star' : 'stars'}`} /><Icon name="star" size={28} filled={star <= value} /></label>)}</fieldset>
  return <span className="star-rating" role="img" aria-label={`${value.toFixed(1)} out of 5 stars`}>{[1, 2, 3, 4, 5].map((star) => <span className="star" key={star}><Icon name="star" size={16} /><span className="star-fill" style={{ width: `${Math.min(1, Math.max(0, value - star + 1)) * 100}%` }}><Icon name="star" size={16} filled /></span></span>)}</span>
}
