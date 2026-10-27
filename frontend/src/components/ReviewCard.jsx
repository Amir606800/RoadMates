import Avatar from './Avatar.jsx'
import StarRating from './StarRating.jsx'
import { formatDate } from '../utils/format.js'

export default function ReviewCard({ review }) {
  return <article className="card review-card stack"><div className="row between"><div className="row"><Avatar name={review.author} tone="peach" /><div><h3>{review.author}</h3><p className="text-sm muted">{review.route}</p></div></div><time dateTime={review.date}>{formatDate(review.date)}</time></div><StarRating value={review.rating} /><p className="review-comment">{review.comment}</p>{review.tags?.length > 0 && <div className="chip-list">{review.tags.map((tag) => <span className="chip" key={tag}>{tag}</span>)}</div>}</article>
}
