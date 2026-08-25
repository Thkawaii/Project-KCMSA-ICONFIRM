import { apiFetch, API_BASE_URL, getToken } from './client.js'

export function getImportLicenseItems({ licenseNo, invoiceNo, status, code } = {}) {
  const params = new URLSearchParams()
  if (licenseNo) params.set('license_no', licenseNo)
  if (invoiceNo) params.set('invoice_no', invoiceNo)
  if (status) params.set('status', status)
  if (code) params.set('code', code)

  const qs = params.toString()
  return apiFetch(`/import-license${qs ? `?${qs}` : ''}`)
}

export function getImportLicenseSummary() {
  return apiFetch('/import-license/summary')
}

export function getImportLicenseAlerts({ withinDays, onlyAlert } = {}) {
  const params = new URLSearchParams()
  if (withinDays) params.set('within_days', String(withinDays))
  if (onlyAlert) params.set('only', 'alert')
  const qs = params.toString()
  return apiFetch(`/import-license/alerts${qs ? `?${qs}` : ''}`)
}

export function verifyImportLicenseCode({ code, invoiceNo = '', productionNo = '' }) {
  return apiFetch('/import-license/verify', {
    method: 'POST',
    body: JSON.stringify({ code, invoiceNo, productionNo }),
  })
}

export function deleteImportLicenseItem(id) {
  return apiFetch(`/import-license/${id}`, { method: 'DELETE' })
}

export function renewImportLicense(licenseNo = '', invoiceNo = '', days = 0) {
  return apiFetch('/import-license/renew', {
    method: 'POST',
    body: JSON.stringify({ licenseNo: licenseNo ?? '', invoiceNo: invoiceNo ?? '', days }),
  })
}

export function clearImportLicense(licenseNo = '', invoiceNo = '', all = false) {
  const params = new URLSearchParams()
  if (all) {
    params.set('all', 'true')
  } else {
    params.set('license_no', licenseNo ?? '')
    params.set('invoice_no', invoiceNo ?? '')
  }
  return apiFetch(`/import-license?${params.toString()}`, { method: 'DELETE' })
}

export async function previewImportLicense(file) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE_URL}/import-license/preview`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: formData,
  })

  const data = await res.json().catch(() => null)
  if (!res.ok) {
    throw new Error(data?.message || `Preview failed (${res.status})`)
  }
  return data
}

export async function uploadImportLicense(file) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE_URL}/import-license/upload`, {
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
