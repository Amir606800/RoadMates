import { appVersion } from '../appVersion.js'
import { createAPIClient } from './httpClient.js'
export { APIError } from './httpClient.js'

export const API_BASE = '/api/v1'
let updateRequired = false
const updateListeners = new Set()
export const getUpdateRequired = () => updateRequired
export function subscribeUpdateRequired(listener) { updateListeners.add(listener); return () => updateListeners.delete(listener) }

export const authClient = createAPIClient({ version: appVersion, onUpdateRequired: () => { updateRequired = true; updateListeners.forEach((listener) => listener()) } })
export const apiRequest = authClient.request
export const getAPIVersion = authClient.version
