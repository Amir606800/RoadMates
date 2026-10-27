export function passengerSeats(capacity) {
  return Array.from({ length: capacity }, (_, index) => ({
    id: `seat-${index + 1}`,
    number: index + 1,
    row: index === 0 ? 0 : 1 + Math.floor((index - 1) / 3),
    column: index === 0 ? 2 : (index - 1) % 3,
  }))
}

export function seatState(seatId, layout, selected = [], owned = []) {
  if (owned.includes(seatId)) return 'owned'
  if (layout.bookedSeatIds.includes(seatId)) return 'booked'
  return selected.includes(seatId) ? 'selected' : 'available'
}

export function availableSeatIds(drive, owned = []) {
  const occupied = new Set([...drive.seatLayout.bookedSeatIds, ...owned])
  return passengerSeats(drive.seatLayout.capacity).map((seat) => seat.id).filter((id) => !occupied.has(id))
}

export function activeReservation(reservations, now = Date.now()) {
  return reservations.find((reservation) => reservation.status === 'confirmed' && new Date(reservation.endsAt).getTime() > now) || null
}
