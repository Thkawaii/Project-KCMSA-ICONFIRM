import { apiFetch, API_BASE_URL, getToken } from './client.js'

// รายการ MFG Assembly ทั้งหมด (ใหม่สุดอยู่บน)
export function getMFGAssemblies() {
  return apiFetch('/mfg-assembly')
}

// สแกน QR ตอนประกอบเสร็จ — ส่ง Machine No + IT Controller No.
// backend จะคำนวณ Status (OK/UNKNOWN/REUSED/DUPLICATE) + Country ให้ แล้วตอบกลับ
//   { row, status, matched, message }
export function scanMFGAssembly({ machineNo, itControllerNo }) {
  return apiFetch('/mfg-assembly/scan', {
    method: 'POST',
    body: JSON.stringify({ machineNo, itControllerNo }),
  })
}

// เพิ่มแถวเอง (นอกเหนือจากสแกน)
export function createMFGAssembly(payload) {
  return apiFetch('/mfg-assembly', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

// แก้ไข 1 แถว (ทุกฟิลด์แก้ได้)
export function updateMFGAssembly(id, payload) {
  return apiFetch(`/mfg-assembly/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

// ลบ 1 แถว
export function deleteMFGAssembly(id) {
  return apiFetch(`/mfg-assembly/${id}`, { method: 'DELETE' })
}

// อัปโหลดรูปถ่ายป้ายยืนยัน ผูกกับแถว MFG id (ย้ายมาจากฝั่ง WH)
// เก็บรูปเป็นหลักฐานเฉย ๆ — คืน { row, saved, message } โดย row.PhotoURL อัปเดตแล้ว
//
// ใช้ fetch ตรง (ไม่ใช่ apiFetch) เพราะเป็น multipart/form-data — apiFetch จะยัด
// Content-Type: application/json ทำให้ multipart boundary หาย backend parse ไฟล์ไม่ได้
export async function uploadMFGAssemblyPhoto(id, fileOrBlob) {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', fileOrBlob, 'mfg-photo.jpg')

  const res = await fetch(`${API_BASE_URL}/mfg-assembly/${id}/photo`, {
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
