import { apiFetch, API_BASE_URL, getToken } from './client.js'

export function getMachineSpecs(componentType) {
  const qs = componentType ? `?component_type=${encodeURIComponent(componentType)}` : ''
  return apiFetch(`/machine-spec${qs}`)
}

export function getMachineSpecDetail(id) {
  return apiFetch(`/machine-spec/${id}`)
}

export function deleteMachineSpec(id) {
  return apiFetch(`/machine-spec/${id}`, { method: 'DELETE' })
}

// ใช้ fetch ตรงแทน apiFetch เพราะเป็น multipart/form-data
// (apiFetch ใส่ Content-Type: application/json ให้อัตโนมัติ ซึ่งจะทำให้
// multipart boundary หายไปและ backend parse ไฟล์ไม่ได้)
export async function uploadMachineSpec(componentType, file) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE_URL}/machine-spec/upload/${componentType}`, {
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

// ดาวน์โหลดไฟล์ export โดยตรง (ต้องแนบ token เอง เพราะเป็น GET แบบไบนารี
// ไม่ใช่ JSON ธรรมดา ใช้ <a href> ตรงๆ ไม่ได้เพราะต้องมี Authorization header)
export async function exportMachineSpecs(componentType) {
  const token = getToken()
  const qs = componentType ? `?component_type=${encodeURIComponent(componentType)}` : ''

  const res = await fetch(`${API_BASE_URL}/machine-spec/export${qs}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })

  if (!res.ok) {
    throw new Error(`Export failed (${res.status})`)
  }

  const blob = await res.blob()
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `master-data-export-${Date.now()}.xlsx`
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.URL.revokeObjectURL(url)
}