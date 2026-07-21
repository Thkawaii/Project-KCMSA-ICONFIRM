import { apiFetch } from './client.js'

export function getPartChecks() {
  return apiFetch('/part-check')
}

export function scanPartCheck(tag) {
  return apiFetch('/part-check', {
    method: 'POST',
    body: JSON.stringify({ tag }),
  })
}