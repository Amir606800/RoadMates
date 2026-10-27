// Reviews remain a UI preview until a review API is introduced.
export async function submitReview(review) {
  await new Promise((resolve) => setTimeout(resolve, 450))
  return { ...review, id: `review-${Date.now()}` }
}
