import { useState } from 'react'
import { submitReview } from '../api/index.js'
import { mockCompletedTrips } from '../mocks/data.js'
import { useApp } from '../context/AppContext.jsx'
import { dateAfter, formatDate } from '../utils/format.js'
import Field from './Field.jsx'
import Icon from './Icon.jsx'
import StarRating from './StarRating.jsx'

export default function ReviewForm() {
  const { user, role, writtenReviews, setWrittenReviews } = useApp()
  const trips = mockCompletedTrips.filter((trip) => trip.role === role && !writtenReviews.some((review) => review.tripId === trip.id))
  const [tripId, setTripId] = useState('')
  const [rating, setRating] = useState(0)
  const [comment, setComment] = useState('')
  const [errors, setErrors] = useState({})
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState('')
  const trip = trips.find((item) => item.id === tripId) || trips[0]
  async function submit(event) {
    event.preventDefault()
    const nextErrors = {}
    if (!rating) nextErrors.rating = 'Choose a rating from 1 to 5 stars.'
    if (comment.trim().length < 5 || comment.trim().length > 280) nextErrors.comment = 'Write a short comment between 5 and 280 characters.'
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length || !trip) return
    setSubmitting(true)
    try {
      const review = await submitReview({ tripId: trip.id, authorRole: role, author: `${user.name} ${user.surname}`, recipient: trip.person, target: role === 'client' ? 'driver' : 'client', rating, comment: comment.trim(), route: `${trip.from} → ${trip.to}`, date: dateAfter(), tags: [] })
      setWrittenReviews((items) => [review, ...items])
      setRating(0); setComment(''); setMessage(`Thanks! Your review of ${trip.person} is saved.`)
    } catch { setErrors({ submit: 'Your review could not be saved. Please try again.' }) }
    finally { setSubmitting(false) }
  }
  return <>
    <div className="section-heading"><h2>Leave a review</h2><Icon name="message" size={18} /></div>
    {message && <div className="notice review-saved" role="status"><Icon name="check" /><p>{message}</p></div>}
    {trip ? <form className="card stack" noValidate onSubmit={submit}><Field id="review-trip" label="Completed journey"><select id="review-trip" value={trip.id} onChange={(event) => { setTripId(event.target.value); setRating(0); setComment(''); setErrors({}); setMessage('') }} disabled={submitting}>{trips.map((item) => <option value={item.id} key={item.id}>{item.person} · {formatDate(item.date)}</option>)}</select></Field><p className="text-sm muted">{trip.from} → {trip.to}{role === 'driver' && ' · Rate punctuality, politeness, and cancellations.'}</p><div><p className="form-label">How was your journey?</p><StarRating value={rating} onChange={(value) => { setRating(value); setErrors({ ...errors, rating: '' }) }} disabled={submitting} />{errors.rating && <p className="field-error" role="alert">{errors.rating}</p>}</div><Field id="review-comment" label="A few words" error={errors.comment}><textarea id="review-comment" value={comment} disabled={submitting} maxLength={280} onChange={(event) => { setComment(event.target.value); setErrors({ ...errors, comment: '' }) }} placeholder={role === 'client' ? 'What made the ride a good one?' : 'Were they on time and easy to travel with?'} aria-invalid={Boolean(errors.comment)} aria-describedby={errors.comment ? 'review-comment-error' : undefined} /></Field>{errors.submit && <p className="field-error" role="alert">{errors.submit}</p>}<button className="button" disabled={submitting}>{submitting ? 'Saving your review…' : 'Share your review'}<Icon name="arrow" size={17} /></button></form> : <div className="card"><h3>You’re all caught up</h3><p className="text-sm muted">Your next completed journey will appear here.</p></div>}
  </>
}
