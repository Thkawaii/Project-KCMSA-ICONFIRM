import { apiFetch, API_BASE_URL, getToken } from './client.js'

// ชนิดไฟล์ที่หน้า Upload Data รองรับ (ต้องตรงกับ backend models.Dataset*)
export const DATASETS = [
  { key: 'planning', label: 'Planning' },
  { key: 'wh1', label: 'WH1' },
  { key: 'wh2', label: 'WH2' },
  { key: 'engine', label: 'Engine' },
]

// คืน { dataset, columns: [...ชื่อคอลัมน์ตามลำดับ], rows: [...UploadDataRow] }
export function getUploadData(dataset, keyword) {
  const params = new URLSearchParams({ dataset })
  if (keyword) params.set('keyword', keyword)
  return apiFetch(`/upload-data?${params.toString()}`)
}

// ใช้ fetch ตรงเพราะเป็น multipart/form-data (apiFetch ยัด Content-Type: application/json)
export async function uploadDataFile(dataset, file) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE_URL}/upload-data/upload/${dataset}`, {
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

export function deleteUploadDataRow(id) {
  return apiFetch(`/upload-data/${id}`, { method: 'DELETE' })
}

export function clearUploadData(dataset) {
  return apiFetch(`/upload-data?dataset=${encodeURIComponent(dataset)}`, { method: 'DELETE' })
}

// ดาวน์โหลดไฟล์ export (GET แบบไบนารี ต้องแนบ token เอง)
export async function exportUploadData(dataset) {
  const token = getToken()
  const res = await fetch(`${API_BASE_URL}/upload-data/export?dataset=${encodeURIComponent(dataset)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })

  if (!res.ok) {
    throw new Error(`Export failed (${res.status})`)
  }

  const blob = await res.blob()
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${dataset}-export-${Date.now()}.xlsx`
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.URL.revokeObjectURL(url)
}
