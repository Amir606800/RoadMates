import { useState } from 'react'
import { useApp } from '../context/AppContext.jsx'
import { mockReviews } from '../mocks/data.js'
import RatingSummary from '../components/RatingSummary.jsx'
import ReviewCard from '../components/ReviewCard.jsx'
import ReviewForm from '../components/ReviewForm.jsx'
import EmptyState from '../components/EmptyState.jsx'

export default function ReviewsPage() {
  const { role, writtenReviews } = useApp()
  const [tab, setTab] = useState('about')
  const reviews = tab === 'about' ? mockReviews.filter((review) => review.target === role) : writtenReviews.filter((review) => review.authorRole === role)
  const secondTab = role === 'client' ? 'Driver reviews' : 'Client behaviour'
  function keyDown(event) {
    if (['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) { event.preventDefault(); const next = event.key === 'Home' ? 'about' : event.key === 'End' ? 'written' : tab === 'about' ? 'written' : 'about'; setTab(next); document.getElementById(`reviews-tab-${next}`)?.focus() }
  }
  return <>
    <section className="page-intro"><span className="eyebrow">A LITTLE TRUST GOES FAR</span><h1>Good words.<br />Better journeys.</h1><p>The people make the difference.</p></section>
    <div className="review-tabs" role="tablist" aria-label="Review categories" onKeyDown={keyDown}>{[['about', 'About me'], ['written', secondTab]].map(([key, label]) => <button id={`reviews-tab-${key}`} key={key} role="tab" aria-selected={tab === key} aria-controls="reviews-panel" tabIndex={tab === key ? 0 : -1} className={tab === key ? 'selected' : ''} onClick={() => setTab(key)}>{label}</button>)}</div>
    <div role="tabpanel" id="reviews-panel" aria-labelledby={`reviews-tab-${tab}`} tabIndex={0}>
      <p className="review-explainer">{tab === 'about' ? role === 'driver' ? 'What clients say about your driving.' : 'What drivers say about your punctuality, politeness, and cancellations.' : role === 'client' ? 'Reviews you’ve shared about your drivers.' : 'Reviews you’ve shared about your passengers.'}</p>
      <RatingSummary reviews={reviews} />
      <div className="section-heading"><h2>{tab === 'about' ? 'From your road mates' : 'Your shared reviews'}</h2><span>{reviews.length} total</span></div>
      <div className="stack">{reviews.length ? reviews.map((review) => <ReviewCard review={review} key={review.id} />) : <EmptyState icon="star" title="Your words belong here">Review a completed journey below. Your feedback helps the next road mate.</EmptyState>}</div>
    </div>
    <ReviewForm />
  </>
}
