export function dateAfter(days = 0) {
  const date = new Date()
  date.setDate(date.getDate() + days)
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Baku', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(date)
  const part = (type) => parts.find((item) => item.type === type).value
  return `${part('year')}-${part('month')}-${part('day')}`
}

export function formatDate(value, options = {}) {
  return new Date(`${value}T12:00:00`).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', ...options })
}

export const money = (value) => `${Number(value).toLocaleString('en-GB', { maximumFractionDigits: 2 })} ₼`
export const normalizeCity = (value) => value.trim().toLocaleLowerCase('en')
