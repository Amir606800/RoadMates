import { test } from 'node:test'
import assert from 'node:assert/strict'
import { validateDrive, validateVerification } from '../src/utils/validation.js'
import { dateAfter } from '../src/utils/format.js'
import { buildRide, rideSearchQuery, rideDate, rideTime } from '../src/api/ridePayload.js'
import { buildDriverApplication } from '../src/api/driverApplicationPayload.js'

const application = { phone: '+994 50 234 56 78', name: 'Leyla', surname: 'Mammadova', pin: 'ABC1234', brand: 'Toyota', model: 'Corolla', year: '2022', color: 'Silver', plate: '10-RM-245', seats: '4' }
const drive = { from: { id: 1, name_az: 'Departure' }, to: { id: 2, name_az: 'Destination' }, date: dateAfter(1), time: '09:00', seats: '3', price: '20', note: '' }

test('valid application accepts international names and a masked-input PIN format', () => {
  assert.deepEqual(validateVerification({ ...application, name: 'Əli', surname: 'Məmmədov' }), {})
})

test('verification validates identity and vehicle fields with no sensitive echo', () => {
  const errors = validateVerification({ ...application, phone: 'invalid', name: '', surname: '1', pin: '<bad>', brand: '', model: '', year: '1900', color: '', plate: '@', seats: '0' })
  for (const key of ['phone', 'name', 'surname', 'pin', 'brand', 'model', 'year', 'color', 'plate', 'seats']) assert.ok(errors[key], `Missing ${key} error`)
  assert.equal(JSON.stringify(errors).includes('<bad>'), false)
})

test('verification rejects malformed PINs and seat/year bounds', () => {
  for (const pin of ['', '123456', '12345678', 'ABC!234', '       ']) assert.ok(validateVerification({ ...application, pin }).pin)
  assert.ok(validateVerification({ ...application, year: '2100' }, 2026).year)
  assert.ok(validateVerification({ ...application, seats: '9' }).seats)
})

test('driver application payload sends the PIN once and whitelists request fields', () => {
  const result = buildDriverApplication({ ...application, pin: 'abc1234', roles: ['admin'], status: 'approved', user_id: 'someone-else' })
  assert.equal(result.pin_code, 'ABC1234')
  assert.equal(result.car_year, 2022)
  assert.equal(result.seats, 4)
  assert.equal(Object.hasOwn(result, 'roles'), false)
  assert.equal(Object.hasOwn(result, 'user_id'), false)
  assert.equal(Object.hasOwn(result, 'status'), false)
  assert.equal(Object.hasOwn(result, 'pin'), false)
  assert.equal(Object.keys(result).length, 10)
})

test('drive validation accepts future trips and rejects identical or unselected districts', () => {
  assert.deepEqual(validateDrive(drive), {})
  assert.ok(validateDrive({ ...drive, to: { id: 1, name_az: 'Other label, same ID' } }).to)
  assert.ok(validateDrive({ ...drive, from: null }).from)
})

test('drive validation rejects past/impossible dates and malformed times', () => {
  assert.ok(validateDrive({ ...drive, date: dateAfter(-1) }).date)
  assert.ok(validateDrive({ ...drive, date: '2099-02-30' }).date)
  assert.ok(validateDrive({ ...drive, time: '25:00' }).time)
  assert.ok(validateDrive({ ...drive, date: dateAfter(), time: '00:00' }).time)
})

test('drive validation enforces meaningful price, seat, and note limits', () => {
  for (const price of ['', '-1', '1000000', 'NaN', '2.001', '1e2']) assert.ok(validateDrive({ ...drive, price }).price)
  assert.deepEqual(validateDrive({ ...drive, price: '0' }), {})
  for (const seats of ['0', '9', '1.5']) assert.ok(validateDrive({ ...drive, seats }).seats)
  assert.ok(validateDrive({ ...drive, note: 'x'.repeat(301) }).note)
})

test('ride payload whitelists IDs and converts Azerbaijan departure to UTC', () => {
  const payload = buildRide({ ...drive, date: '2030-01-02', price: '20.25', note: '  Meeting point  ', driver_id: 'someone-else', status: 'completed' })
  assert.deepEqual(payload, { from_district_id: 1, to_district_id: 2, departure_at: '2030-01-02T05:00:00.000Z', seats_total: 3, price_azn: 20.25, notes: 'Meeting point' })
})

test('ride search sends database IDs and date, never district labels', () => {
  assert.equal(rideSearchQuery({ ...drive, date: '2030-01-02' }), 'from_district_id=1&to_district_id=2&date=2030-01-02')
  assert.equal(rideSearchQuery({}), '')
})

test('ride date and time use Azerbaijan midnight boundaries', () => {
  assert.equal(rideDate('2030-01-01T21:30:00Z'), '2030-01-02')
  assert.equal(rideTime('2030-01-01T21:30:00Z'), '01:30')
})
