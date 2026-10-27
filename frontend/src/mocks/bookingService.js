import { activeReservation, availableSeatIds } from '../utils/seats.js'

// Session-only mock authority. The real API must enforce the same checks in
// an authenticated database transaction; frontend checks alone are insufficient.
export function createBookingService({ findDrive, wait = async () => {}, now = Date.now }) {
  const reservations = []
  const pendingUsers = new Set()

  function currentBooking(userId) {
    return activeReservation(reservations.filter((item) => item.userId === userId), now())
  }

  function bookedSeatIds(driveId) {
    return reservations.filter((item) => item.driveId === driveId && item.status === 'confirmed')
      .flatMap((item) => item.seatIds)
  }

  async function reserve(userId, driveId, seatIds) {
    if (!userId) throw new Error('Sign in before reserving seats.')
    if (pendingUsers.has(userId)) throw new Error('Your booking is already being processed. Please wait.')
    if (currentBooking(userId)) throw new Error('You already have an active booking. You can book another car after that journey ends.')
    const drive = findDrive(driveId)
    if (!drive) throw new Error('This drive is no longer available.')
    if (!Array.isArray(seatIds) || !seatIds.length || new Set(seatIds).size !== seatIds.length) throw new Error('Select at least one available seat, without duplicates.')
    const endsAt = `${drive.date}T${drive.arrival}:00`
    if (new Date(`${drive.date}T${drive.time}:00`).getTime() <= now()) throw new Error('This drive has already departed.')
    const requested = [...seatIds]
    function checkSeats() {
      const available = availableSeatIds(drive, bookedSeatIds(driveId))
      if (requested.some((id) => !available.includes(id))) throw new Error('One of these seats is no longer available. Please choose another seat.')
    }
    checkSeats()
    pendingUsers.add(userId)
    try {
      await wait()
      if (new Date(`${drive.date}T${drive.time}:00`).getTime() <= now()) throw new Error('This drive has already departed.')
      // Recheck after waiting: another client may have booked the same seat.
      checkSeats()
      const reservation = {
        id: `reservation-${now()}-${reservations.length + 1}`, userId, driveId,
        seats: requested.length, seatIds: requested, status: 'confirmed', endsAt,
      }
      reservations.push(reservation)
      return reservation
    } finally {
      pendingUsers.delete(userId)
    }
  }

  return { reserve, currentBooking, bookedSeatIds }
}
