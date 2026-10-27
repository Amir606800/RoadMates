// IDs come from the districts endpoint; names never become API identifiers.
export function rideSearchQuery(filters) {
  const query = new URLSearchParams()
  if (filters.from?.id) query.set('from_district_id', filters.from.id)
  if (filters.to?.id) query.set('to_district_id', filters.to.id)
  if (filters.date) query.set('date', filters.date)
  return query.toString()
}

export function buildRide(values) {
  return {
    from_district_id: values.from.id, to_district_id: values.to.id,
    departure_at: new Date(`${values.date}T${values.time}:00+04:00`).toISOString(),
    seats_total: Number(values.seats), price_azn: Number(values.price), notes: values.note.trim(),
  }
}

export function rideDate(value) {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Baku', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date(value))
  const part = (type) => parts.find((item) => item.type === type).value
  return `${part('year')}-${part('month')}-${part('day')}`
}
export const rideTime = (value) => new Date(value).toLocaleTimeString('en-GB', { timeZone: 'Asia/Baku', hour: '2-digit', minute: '2-digit' })
export const driverName = (ride) => `${ride.driver.first_name} ${ride.driver.last_name}`
