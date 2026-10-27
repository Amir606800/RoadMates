import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createBookingService } from '../src/mocks/bookingService.js'
import { passengerSeats, seatState, availableSeatIds } from '../src/utils/seats.js'

const drive = { id: 'car-a', date: '2099-01-01', time: '09:00', arrival: '13:00', seatLayout: { capacity: 4, bookedSeatIds: ['seat-4'] } }
function service(options = {}) {
  return createBookingService({ findDrive: (id) => ['car-a', 'car-b'].includes(id) ? { ...drive, id } : null, ...options })
}

test('seat positions match capacity and keep the driver separate', () => {
  for (const capacity of [1, 4, 6, 8]) {
    const seats = passengerSeats(capacity)
    assert.equal(seats.length, capacity)
    assert.equal(new Set(seats.map((seat) => `${seat.row}:${seat.column}`)).size, capacity)
    assert.ok(seats.every((seat) => seat.row !== 0 || seat.column !== 0))
  }
})

test('gray/yellow/black/green states prioritize ownership and occupied seats', () => {
  assert.equal(seatState('seat-1', drive.seatLayout), 'available')
  assert.equal(seatState('seat-1', drive.seatLayout, ['seat-1']), 'selected')
  assert.equal(seatState('seat-4', drive.seatLayout, ['seat-4']), 'booked')
  assert.equal(seatState('seat-4', drive.seatLayout, [], ['seat-4']), 'owned')
  assert.deepEqual(availableSeatIds(drive, ['seat-1']), ['seat-2', 'seat-3'])
})

test('confirms exact seat IDs together in one car', async () => {
  const booking = await service().reserve('client', 'car-a', ['seat-1', 'seat-2'])
  assert.deepEqual(booking.seatIds, ['seat-1', 'seat-2'])
  assert.equal(booking.seats, 2)
  assert.equal(booking.status, 'confirmed')
})

test('rejects a second active booking in another car or the same car', async () => {
  const bookings = service()
  await bookings.reserve('client', 'car-a', ['seat-1'])
  await assert.rejects(bookings.reserve('client', 'car-b', ['seat-2']), /active booking/)
  await assert.rejects(bookings.reserve('client', 'car-a', ['seat-2']), /active booking/)
})

test('simultaneous requests by one client can confirm only one car', async () => {
  const bookings = service({ wait: () => new Promise((resolve) => setTimeout(resolve, 10)) })
  const results = await Promise.allSettled([bookings.reserve('client', 'car-a', ['seat-1']), bookings.reserve('client', 'car-b', ['seat-1'])])
  assert.equal(results.filter((result) => result.status === 'fulfilled').length, 1)
  assert.match(results.find((result) => result.status === 'rejected').reason.message, /already being processed/)
})

test('two clients cannot purchase the same seat concurrently', async () => {
  const bookings = service({ wait: () => new Promise((resolve) => setTimeout(resolve, 10)) })
  const results = await Promise.allSettled([bookings.reserve('client-a', 'car-a', ['seat-1']), bookings.reserve('client-b', 'car-a', ['seat-1'])])
  assert.equal(results.filter((result) => result.status === 'fulfilled').length, 1)
  assert.match(results.find((result) => result.status === 'rejected').reason.message, /no longer available/)
})

test('rejects occupied, unknown, duplicate, or empty seat requests', async () => {
  const bookings = service()
  for (const seats of [[], ['seat-4'], ['seat-99'], ['seat-1', 'seat-1']]) await assert.rejects(bookings.reserve('client', 'car-a', seats))
  await assert.rejects(bookings.reserve('client', 'missing', ['seat-1']), /no longer available/)
  await assert.rejects(bookings.reserve('', 'car-a', ['seat-1']), /Sign in/)
  assert.equal((await bookings.reserve('client', 'car-a', ['seat-1'])).seats, 1)
})

test('a failed service request releases the booking lock for retry', async () => {
  let shouldFail = true
  const bookings = service({ wait: async () => { if (shouldFail) throw new Error('Network failure') } })
  await assert.rejects(bookings.reserve('client', 'car-a', ['seat-1']), /Network failure/)
  shouldFail = false
  assert.equal((await bookings.reserve('client', 'car-b', ['seat-1'])).driveId, 'car-b')
})

test('after a journey ends a client can book another future car', async () => {
  let clock = new Date('2099-01-01T08:00:00').getTime()
  const bookings = service({ now: () => clock, findDrive: (id) => ({ ...drive, id, date: id === 'car-a' ? '2099-01-01' : '2099-01-02' }) })
  await bookings.reserve('client', 'car-a', ['seat-1'])
  clock = new Date('2099-01-01T13:01:00').getTime()
  assert.equal(bookings.currentBooking('client'), null)
  assert.equal((await bookings.reserve('client', 'car-b', ['seat-2'])).driveId, 'car-b')
})

test('does not confirm a departed drive, including departure during processing', async () => {
  let clock = new Date('2099-01-01T09:00:00').getTime()
  const bookings = service({ now: () => clock, wait: async () => { clock = new Date('2099-01-01T09:00:00').getTime() } })
  await assert.rejects(bookings.reserve('client', 'car-a', ['seat-1']), /already departed/)
  clock = new Date('2099-01-01T08:59:59').getTime()
  await assert.rejects(bookings.reserve('client', 'car-a', ['seat-1']), /already departed/)
  assert.equal(bookings.currentBooking('client'), null)
})
