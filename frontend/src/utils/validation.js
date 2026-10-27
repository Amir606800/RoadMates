import { dateAfter } from './format.js'

export function validateVerification(values, currentYear = new Date().getFullYear()) {
  const errors = {}
  if (values.phone.trim().length > 25 || !/^\+?[\d ()-]+$/.test(values.phone.trim()) || !/^\d{7,15}$/.test(values.phone.replace(/\D/g, ''))) errors.phone = 'Enter a valid phone number with 7–15 digits.'
  for (const key of ['name', 'surname']) {
    if (!/^[\p{L}][\p{L}\p{M} '’-]{0,59}$/u.test(values[key].trim())) errors[key] = `Enter your ${key} (1–60 letters).`
  }
  if (!/^[a-z0-9]{7}$/i.test(values.pin.trim())) errors.pin = 'Enter the 7 letters and numbers of your PIN code.'
  for (const key of ['brand', 'model', 'color']) {
    if (!values[key].trim() || [...values[key].trim()].length > 50 || /\p{Cc}/u.test(values[key])) errors[key] = `Enter a car ${key} (up to 50 characters).`
  }
  if (!/^\d{4}$/.test(values.year) || Number(values.year) < 1980 || Number(values.year) > currentYear + 1) errors.year = `Enter a year from 1980 to ${currentYear + 1}.`
  if (!/^[\p{L}\p{N}][\p{L}\p{N} -]{2,14}$/u.test(values.plate.trim())) errors.plate = 'Enter a valid plate number (3–15 letters or numbers).'
  if (!/^[1-8]$/.test(String(values.seats))) errors.seats = 'Enter 1–8 passenger seats.'
  return errors
}

export function validateDrive(values) {
  const errors = {}
  for (const key of ['from', 'to']) if (!Number.isInteger(values[key]?.id) || values[key].id < 1 || values[key].id > 32767) errors[key] = 'Choose a district.'
  if (values.from && values.from.id === values.to?.id) errors.to = 'Choose a different destination.'
  const parsedDate = new Date(`${values.date}T12:00:00`)
  const isRealDate = !Number.isNaN(parsedDate.getTime()) && `${parsedDate.getFullYear()}-${String(parsedDate.getMonth() + 1).padStart(2, '0')}-${String(parsedDate.getDate()).padStart(2, '0')}` === values.date
  if (!/^\d{4}-\d{2}-\d{2}$/.test(values.date) || values.date < dateAfter() || !isRealDate) errors.date = 'Choose today or a future date.'
  if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(values.time)) errors.time = 'Choose a departure time.'
  else if (new Date(`${values.date}T${values.time}:00+04:00`) <= new Date()) errors.time = 'Choose a time in the future.'
  if (!/^[1-8]$/.test(String(values.seats))) errors.seats = 'Offer 1–8 passenger seats.'
  if (!/^(0|[1-9]\d{0,5})(\.\d{1,2})?$/.test(String(values.price))) errors.price = 'Enter 0–999999.99 AZN with up to 2 decimals.'
  if ([...values.note].length > 300 || /[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F]/u.test(values.note)) errors.note = 'Use up to 300 characters without control characters.'
  return errors
}
