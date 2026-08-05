import { apiFetch, API_BASE_URL, getToken } from './client.js'

// getPartChecks() -> ประวัติสแกนทั้งหมด
// getPartChecks({ invoiceNo: 'TQ60610' }) -> เฉพาะล็อตนั้น
export function getPartChecks({ invoiceNo, partType } = {}) {
  const params = new URLSearchParams()
  if (invoiceNo) params.set('invoice_no', invoiceNo)
  if (partType) params.set('part_type', partType)

  const qs = params.toString()
  return apiFetch(`/part-check${qs ? `?${qs}` : ''}`)
}

// บันทึกผลสแกน 1 รายการ
//
// สำหรับพาร์ทชนิด ITC backend จะเทียบ sn (หมายเลขเครื่อง 12 หลัก) กับบัญชี
// ใบอนุญาตนำเข้าให้ทันที แล้วตอบกลับมาเป็น
//   { check, matchStatus, matched, message, item }
// โดย item คือแถวในบัญชีที่จับคู่ได้ (null ถ้าไม่เจอ)
export function scanPartCheck({ machineTag, partType, pn, sn, productionNo = '', invoiceNo = '' }) {
  return apiFetch('/part-check', {
    method: 'POST',
    body: JSON.stringify({ machineTag, partType, pn, sn, productionNo, invoiceNo }),
  })
}

// ลบรายการประวัติการสแกน — backend อนุญาตเฉพาะรายการที่ "ไม่พบในใบอนุญาต" (NOT_FOUND)
export function deletePartCheck(id) {
  return apiFetch(`/part-check/${id}`, { method: 'DELETE' })
}

// อัปโหลดรูปถ่ายป้ายยืนยัน ผูกกับรายการสแกน id — backend จะเรียก Claude Vision
// อ่าน P/N, S/N, IMEI จากรูป มาเทียบกับค่าที่สแกนไว้แล้วตอบกลับ
//   { check, matched, message }
// check คือ PartCheck ที่อัปเดต PhotoURL / PhotoOCR* / PhotoMatchStatus แล้ว
//
// ใช้ fetch ตรงแทน apiFetch เพราะเป็น multipart/form-data (apiFetch ใส่
// Content-Type: application/json ให้อัตโนมัติ ซึ่งจะทำให้ multipart boundary
// หายไปและ backend parse ไฟล์ไม่ได้)
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
