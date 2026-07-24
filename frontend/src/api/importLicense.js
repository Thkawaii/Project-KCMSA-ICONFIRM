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

export function clearImportLicense(licenseNo) {
  return apiFetch(`/import-license?license_no=${encodeURIComponent(licenseNo)}`, {
    method: 'DELETE',
  })
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
