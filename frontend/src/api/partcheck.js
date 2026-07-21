import { apiFetch } from './client.js'

export function getPartChecks() {
  return apiFetch('/part-check')
}

export function scanPartCheck({ machineTag, partType, pn, sn }) {
  return apiFetch('/part-check', {
    method: 'POST',
    body: JSON.stringify({ machineTag, partType, pn, sn }),
  })
}