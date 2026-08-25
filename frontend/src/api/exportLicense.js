import { apiFetch, API_BASE_URL, getToken } from './client.js'

export function getExportLicense(q = '', link = '') {
  const params = new URLSearchParams()
  if (q) params.set('q', q)
  if (link === 'matched' || link === 'unmatched') params.set('link', link)
  const qs = params.toString()
  return apiFetch(`/export-license${qs ? `?${qs}` : ''}`)
}

export function getExportLicenseTrace(id) {
  return apiFetch(`/export-license/${id}/trace`)
}

export function getExportLicenseAlerts({ withinDays, onlyAlert } = {}) {
  const params = new URLSearchParams()
  if (withinDays) params.set('within_days', String(withinDays))
  if (onlyAlert) params.set('only', 'alert')
  const qs = params.toString()
  return apiFetch(`/export-license/alerts${qs ? `?${qs}` : ''}`)
}

export function deleteExportLicense(id) {
  return apiFetch(`/export-license/${id}`, { method: 'DELETE' })
}

export function clearExportLicense(licenseNo = '', all = false) {
  const params = new URLSearchParams()
  if (all || !licenseNo) {
    params.set('all', 'true')
  } else {
    params.set('license_no', licenseNo)
  }
  return apiFetch(`/export-license?${params.toString()}`, { method: 'DELETE' })
}

export function renewExportLicense(exportLicenseNo = '', invoiceNo = '', days = 0) {
  return apiFetch('/export-license/renew', {
    method: 'POST',
    body: JSON.stringify({ exportLicenseNo, invoiceNo, days }),
  })
}

export async function previewExportLicense(file) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE_URL}/export-license/preview`, {
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

export async function uploadExportLicense(file) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE_URL}/export-license/upload`, {
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
