import { apiFetch } from './client.js'

export function getQaQueue() {
  return apiFetch('/qa')
}

export function confirmQaResult(id, result, remark = '') {
  return apiFetch(`/qa-confirm/${id}`, {
    method: 'POST',
    body: JSON.stringify({ result, remark }),
  })
}

export function getAuditLog() {
  return apiFetch('/audit-log')
}