import { apiFetch } from './client.js'

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
