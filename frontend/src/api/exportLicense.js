import { apiFetch, API_BASE_URL, getToken } from './client.js'

// ── บัญชีใบอนุญาตส่งออก (คู่กับ Import License) ──────────────────────────────
// อัปโหลดไฟล์ Excel/CSV ที่มีคอลัมน์ ใบขน (Date) / Exception License /
// Serial Number / Expire date เก็บไว้เป็น "ตารางอ้างอิง" ฝั่งขาออก
// หลักการเดียวกับ importLicense.js / whStock.js

// getExportLicense() -> ทั้งหมด (แต่ละแถวมี field `Link` = ผลการเชื่อมโยง)
// getExportLicense('YN15436814') -> ค้น Serial / Exception / Machine No / IT Controller No / Invoice No
// getExportLicense('', 'matched'|'unmatched') -> กรองตามสถานะการเชื่อม
export function getExportLicense(q = '', link = '') {
  const params = new URLSearchParams()
  if (q) params.set('q', q)
  if (link === 'matched' || link === 'unmatched') params.set('link', link)
  const qs = params.toString()
  return apiFetch(`/export-license${qs ? `?${qs}` : ''}`)
}

// ลากเส้นทางของ 1 แถวใบอนุญาตส่งออกแบบละเอียด (เปิดใน modal)
// คืน { item, keys, importLicense?, mfgAssembly?, machineSpecs?, whStock? }
export function getExportLicenseTrace(id) {
  return apiFetch(`/export-license/${id}/trace`)
}

// การแจ้งเตือนอายุใบอนุญาตส่งออก (อายุ 1 เดือน)
//   getExportLicenseAlerts()                    -> ทั้งหมด (VALID/EXPIRING/EXPIRED/NO_DATE)
//   getExportLicenseAlerts({ onlyAlert: true }) -> เฉพาะที่หมดอายุ/ใกล้หมดอายุ (ป้อน badge กระดิ่ง)
//   getExportLicenseAlerts({ withinDays: 7 })   -> เปลี่ยนเกณฑ์ "ใกล้หมดอายุ" (ปริยาย 7 วัน)
// คืน { generatedAt, withinDays, counts:{expired,expiring,valid,noDate,alert}, items:[...] }
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

export function clearExportLicense() {
  return apiFetch('/export-license', { method: 'DELETE' })
}

// ใช้ fetch ตรงแทน apiFetch เพราะเป็น multipart/form-data
// (apiFetch ใส่ Content-Type: application/json ให้อัตโนมัติ ซึ่งจะทำให้
// multipart boundary หายไปและ backend parse ไฟล์ไม่ได้)
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
