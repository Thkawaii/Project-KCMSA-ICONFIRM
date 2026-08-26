import { apiFetch, API_BASE_URL, getToken } from './client.js'

async function uploadFile(path, file) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE_URL}${path}`, {
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

export function getWHMachineStock(q = '') {
  const qs = q ? `?q=${encodeURIComponent(q)}` : ''
  return apiFetch(`/wh-stock/mc${qs}`)
}
export function uploadWHMachineStock(file) {
  return uploadFile('/wh-stock/mc/upload', file)
}
export function deleteWHMachineStock(id) {
  return apiFetch(`/wh-stock/mc/${id}`, { method: 'DELETE' })
}
export function clearWHMachineStock() {
  return apiFetch('/wh-stock/mc', { method: 'DELETE' })
}
