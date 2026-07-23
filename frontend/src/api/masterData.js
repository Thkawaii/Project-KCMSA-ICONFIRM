import { apiFetch, API_BASE_URL, getToken } from './client.js'

// getMasterData() -> ทั้งหมด
// getMasterData({ componentType: 'it_controller' }) -> เฉพาะ IT Controller
// getMasterData({ code: 'KQ3000045093' }) -> ยิงค่าที่สแกนได้มาค่าเดียว
//   backend จะไล่เทียบให้ทั้ง S/N, IT Controller no., IMEI และ P/N
export function getMasterData({ componentType, code } = {}) {
  const params = new URLSearchParams()
  if (componentType) params.set('component_type', componentType)
  if (code) params.set('code', code)

  const qs = params.toString()
  return apiFetch(`/master-data${qs ? `?${qs}` : ''}`)
}

export function deleteMasterData(id) {
  return apiFetch(`/master-data/${id}`, { method: 'DELETE' })
}

// ใช้ fetch ตรงแทน apiFetch เพราะเป็น multipart/form-data
// (apiFetch ใส่ Content-Type: application/json ให้อัตโนมัติ ซึ่งจะทำให้
// multipart boundary หายไปและ backend parse ไฟล์ไม่ได้)
export async function uploadMasterData(file, componentType) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)
  if (componentType) formData.append('component_type', componentType)

  const res = await fetch(`${API_BASE_URL}/master-data/upload`, {
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
