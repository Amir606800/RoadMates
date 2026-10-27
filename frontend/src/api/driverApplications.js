import { apiRequest } from './client.js'

export async function getMyApplication(options) {
  try { return await apiRequest('/driver-applications/me', options) }
  catch (error) { if (error.status === 404 && error.code === 'NOT_FOUND') return null; throw error }
}
export const applyToDrive = (application) => apiRequest('/driver-applications', { method: 'POST', body: JSON.stringify(application) })
export const listDriverApplications = (status, offset = 0, options = {}) => apiRequest(`/admin/driver-applications?${new URLSearchParams({ status, offset, limit: 20 })}`, options)
export const getAdminApplication = (id, options) => apiRequest(`/admin/driver-applications/${encodeURIComponent(id)}`, options)
export const reviewDriverApplication = (id, action, reason = '') => apiRequest(`/admin/driver-applications/${encodeURIComponent(id)}`, { method: 'PATCH', body: JSON.stringify({ action, reason: reason.trim() }) })
