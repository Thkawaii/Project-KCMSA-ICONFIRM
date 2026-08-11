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

// ลบทะเบียนกลาง — ระบุ componentType เพื่อลบเฉพาะชนิด หรือ all=true เพื่อลบทั้งหมด
export function clearMasterData({ componentType, all = false } = {}) {
  const params = new URLSearchParams()
  if (all) params.set('all', 'true')
  else if (componentType) params.set('component_type', componentType)
  return apiFetch(`/master-data?${params.toString()}`, { method: 'DELETE' })
}

// ตรวจไฟล์ก่อนอัปโหลดจริง (ตรวจจับการเปลี่ยนข้อมูล) — คืน
//   { headerFound, headerRow, matched[], extra[], skipped, problems[],
//     summary:{ total,new,updated,changed,unchanged }, rows:[{serial,status,diffs[]}] }
export async function previewMasterDataChanges(file, componentType) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)
  if (componentType) formData.append('component_type', componentType)

  const res = await fetch(`${API_BASE_URL}/master-data/preview`, {
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

// แก้ไขทะเบียนกลาง 1 รายการ (ใช้ตอนหน้างานเปลี่ยน format ของ P/N / S/N / Machine No.)
// ส่งเฉพาะฟิลด์ที่ต้องการแก้ก็ได้ ฟิลด์ที่ไม่ส่งจะคงค่าเดิมไว้
//   patch = { PartNo?, SerialNo?, Name?, Model?, ComponentType?, ITControllerNo?, IMEI?, SpecCode?, ItemNo? }
export function updateMasterData(id, patch) {
  return apiFetch(`/master-data/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(patch),
  })
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
