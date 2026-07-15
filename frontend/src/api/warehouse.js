import { apiFetch } from './client.js'

export function getWarehouseStock() {
  return apiFetch('/warehouse')
}

export function issueWarehouseStock(id, qty, remark = '') {
  return apiFetch(`/warehouse/${id}/issue`, {
    method: 'POST',
    body: JSON.stringify({ qty, remark }),
  })
}