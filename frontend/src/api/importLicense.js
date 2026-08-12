import { apiFetch, API_BASE_URL, getToken } from './client.js'

// ── บัญชีแสดงหมายเลขเครื่องแนบท้ายใบอนุญาตนำเข้า ─────────────────────────────
// ตารางนี้คือ "ตัวอ้างอิง" ที่หน้า Part Confirmation เอาค่าที่สแกนได้มาเทียบ
// (หลักการเดียวกับ Master Data ของฝั่ง TSF/QA)

// getImportLicenseItems() -> ทั้งหมด
// getImportLicenseItems({ invoiceNo: 'TQ60610' }) -> เฉพาะล็อตนั้น
// getImportLicenseItems({ code: '878250022501' }) -> ยิงค่าที่สแกนได้มาค่าเดียว
//   backend จะไล่เทียบให้ทั้งหมายเลขเครื่องและหมายเลขการผลิต
export function getImportLicenseItems({ licenseNo, invoiceNo, status, code } = {}) {
  const params = new URLSearchParams()
  if (licenseNo) params.set('license_no', licenseNo)
  if (invoiceNo) params.set('invoice_no', invoiceNo)
  if (status) params.set('status', status)
  if (code) params.set('code', code)

  const qs = params.toString()
  return apiFetch(`/import-license${qs ? `?${qs}` : ''}`)
}

// สรุปรายใบอนุญาต/อินวอยซ์ ว่ามีกี่เครื่อง ยืนยันไปแล้วกี่เครื่อง
export function getImportLicenseSummary() {
  return apiFetch('/import-license/summary')
}

// การแจ้งเตือนอายุใบอนุญาต (อายุ 6 เดือนนับจากวันที่ออก)
//   getImportLicenseAlerts()                        -> ทั้งหมด (VALID/EXPIRING/EXPIRED/NO_DATE)
//   getImportLicenseAlerts({ onlyAlert: true })     -> เฉพาะที่หมดอายุ/ใกล้หมดอายุ (ป้อน badge กระดิ่ง)
//   getImportLicenseAlerts({ withinDays: 14 })      -> เปลี่ยนเกณฑ์ "ใกล้หมดอายุ"
// คืน { generatedAt, withinDays, counts:{expired,expiring,valid,noDate,alert}, items:[...] }
export function getImportLicenseAlerts({ withinDays, onlyAlert } = {}) {
  const params = new URLSearchParams()
  if (withinDays) params.set('within_days', String(withinDays))
  if (onlyAlert) params.set('only', 'alert')
  const qs = params.toString()
  return apiFetch(`/import-license/alerts${qs ? `?${qs}` : ''}`)
}

// เทียบค่าที่สแกนได้กับบัญชี "อย่างเดียว" ไม่บันทึกอะไร
// -> { status, matched, message, item }
export function verifyImportLicenseCode({ code, invoiceNo = '', productionNo = '' }) {
  return apiFetch('/import-license/verify', {
    method: 'POST',
    body: JSON.stringify({ code, invoiceNo, productionNo }),
  })
}

export function deleteImportLicenseItem(id) {
  return apiFetch(`/import-license/${id}`, { method: 'DELETE' })
}

// ต่ออายุใบอนุญาตทั้งล็อต (คู่ เลขใบอนุญาต+อินวอยซ์) — เลื่อนวันหมดอายุออกไป N วัน
// backend จะบวก N วันเข้า IssueDate ของทุกเครื่องในล็อต -> วันหมดอายุ (IssueDate+6เดือน)
// เลื่อนตาม N วัน คำนวณใหม่แบบ realtime พอ client โหลดข้อมูลใหม่
//   คืน { renewed, days, newExpiry }
export function renewImportLicense(licenseNo = '', invoiceNo = '', days = 0) {
  return apiFetch('/import-license/renew', {
    method: 'POST',
    body: JSON.stringify({ licenseNo: licenseNo ?? '', invoiceNo: invoiceNo ?? '', days }),
  })
}

// ลบ "ล็อต" ออกทั้งใบ — เจาะจงด้วยคู่ (เลขใบอนุญาต, อินวอยซ์) ตามที่ UI จัดกลุ่มไว้
// ส่งทั้งสอง key เสมอ (แม้ค่าว่าง) เพื่อรองรับล็อตที่ไม่มีเลขใบอนุญาต/อินวอยซ์
//   clearImportLicense('E0503...', 'TQ60610')  -> ลบเฉพาะล็อตนั้น
//   clearImportLicense('', '')                 -> ลบล็อตที่ไม่มีเลขใบอนุญาต+อินวอยซ์
//   clearImportLicense(null, null, true)       -> ลบทั้งหมด (all=true)
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

// ลองอ่านหัวตารางก่อนอัปโหลดจริง (ไม่บันทึกอะไร)
// -> { file, headerFound, headerRow, matched:[...], extra:[...] }
//    matched = คอลัมน์ที่ระบบแม็ปได้, extra = คอลัมน์ใหม่ที่ระบบยังไม่รู้จัก
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

// ใช้ fetch ตรงแทน apiFetch เพราะเป็น multipart/form-data
// (apiFetch ใส่ Content-Type: application/json ให้อัตโนมัติ ซึ่งจะทำให้
// multipart boundary หายไปและ backend parse ไฟล์ไม่ได้)
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
