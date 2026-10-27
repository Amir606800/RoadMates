import { apiRequest } from './client.js'
import { rideSearchQuery, buildRide } from './ridePayload.js'

export const listDistricts = async (q, options) => (await apiRequest(`/districts?${new URLSearchParams({ q })}`, options)).districts
export const listDrives = async (filters, options) => (await apiRequest(`/rides?${rideSearchQuery(filters)}`, options)).rides
export const getDrive = (id, options) => apiRequest(`/rides/${encodeURIComponent(id)}`, options)
export const createDrive = (values) => apiRequest('/rides', { method: 'POST', body: JSON.stringify(buildRide(values)) })
export async function getActiveDrive(options) {
  try { return await apiRequest('/rides/me/active', options) }
  catch (error) { if (error.status === 404 && error.code === 'NOT_FOUND') return null; throw error }
}
export const finishDrive = (id, action) => apiRequest(`/rides/${encodeURIComponent(id)}/${action}`, { method: 'PATCH' })
