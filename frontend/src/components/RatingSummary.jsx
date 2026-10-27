import StarRating from './StarRating.jsx'

export default function RatingSummary({ reviews }) {
  const average = reviews.length ? reviews.reduce((sum, review) => sum + review.rating, 0) / reviews.length : 0
  return <section className="card rating-summary" aria-label="Rating summary"><div className="rating-total"><strong>{reviews.length ? average.toFixed(1) : '—'}</strong><StarRating value={average} /><span>{reviews.length} {reviews.length === 1 ? 'review' : 'reviews'}</span></div><div className="rating-breakdown">{[5, 4, 3, 2, 1].map((star) => {
    const count = reviews.filter((review) => review.rating === star).length
    return <div className="rating-bar-row" key={star}><span>{star}</span><div className="rating-bar" role="img" aria-label={`${star} stars: ${count} reviews`}><span style={{ width: `${reviews.length ? count / reviews.length * 100 : 0}%` }} /></div><span>{count}</span></div>
  })}</div></section>
}
