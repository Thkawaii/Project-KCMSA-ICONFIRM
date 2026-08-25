import { apiFetch, API_BASE_URL, getToken } from './client.js'

export function getPartChecks({ invoiceNo, partType } = {}) {
  const params = new URLSearchParams()
  if (invoiceNo) params.set('invoice_no', invoiceNo)
  if (partType) params.set('part_type', partType)

  const qs = params.toString()
  return apiFetch(`/part-check${qs ? `?${qs}` : ''}`)
}

export function scanPartCheck({ machineTag, partType, pn, sn, productionNo = '', invoiceNo = '' }) {
  return apiFetch('/part-check', {
    method: 'POST',
    body: JSON.stringify({ machineTag, partType, pn, sn, productionNo, invoiceNo }),
  })
}

export function deletePartCheck(id) {
  return apiFetch(`/part-check/${id}`, { method: 'DELETE' })
}

export async function uploadPartCheckPhoto(id, fileOrBlob) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', fileOrBlob, 'partcheck-photo.jpg')

  const res = await fetch(`${API_BASE_URL}/part-check/${id}/photo`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: formData,
  })

  const data = await res.json().catch(() => null)

  if (!res.ok) {
    throw new Error(data?.message || `Upload failed (${res.status})`)
  }

  return data
}
