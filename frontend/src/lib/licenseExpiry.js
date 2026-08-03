// ─────────────────────────────────────────────────────────────────────────────
// ตัวช่วยคำนวณอายุใบอนุญาตนำเข้าฝั่ง frontend
//
// ใบอนุญาตนำเข้า กสทช. มีอายุ 6 เดือนนับจากวันที่ออก (IssueDate)
// ตรรกะตรงกับ backend (GetImportLicenseAlerts) เป๊ะ ๆ เพื่อให้ป้ายสถานะในตาราง
// กับตัวเลขบนกระดิ่งไม่ขัดกันเอง — เปลี่ยนเกณฑ์ที่ไหนต้องเปลี่ยนให้ตรงกันทั้งสองที่
// ─────────────────────────────────────────────────────────────────────────────

export const LICENSE_VALIDITY_MONTHS = 6

export const EXPIRY_STATUS = {
  EXPIRED: 'EXPIRED', // เลยวันหมดอายุแล้ว
  EXPIRING: 'EXPIRING', // ใกล้หมดอายุ (ภายใน withinDays)
  VALID: 'VALID', // ยังไม่ใกล้หมดอายุ
  NO_DATE: 'NO_DATE', // ยังไม่ได้ระบุวันที่ออกใบอนุญาต
}

// เที่ยงคืนของวันนั้น — ใช้ตัดเวลาออกเพื่อให้ "วันคงเหลือ" คงที่ทั้งวัน
function atMidnight(d) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

// รับ IssueDate (string ISO จาก backend หรือ Date) แล้วคืนข้อมูลอายุครบชุด
//   { hasDate, issueDate, expiryDate, daysLeft, status }
// daysLeft ติดลบ = เลยกำหนดมาแล้วกี่วัน
export function computeLicenseExpiry(issueDateRaw, withinDays = 30) {
  if (!issueDateRaw) {
    return { hasDate: false, issueDate: null, expiryDate: null, daysLeft: null, status: EXPIRY_STATUS.NO_DATE }
  }

  const issue = new Date(issueDateRaw)
  if (Number.isNaN(issue.getTime())) {
    return { hasDate: false, issueDate: null, expiryDate: null, daysLeft: null, status: EXPIRY_STATUS.NO_DATE }
  }

  // +6 เดือน (JS จัดการวันสิ้นเดือนให้เอง เช่น 31 ส.ค. + 6 เดือน)
  const expiry = new Date(issue)
  expiry.setMonth(expiry.getMonth() + LICENSE_VALIDITY_MONTHS)

  const today = atMidnight(new Date())
  const expDay = atMidnight(expiry)
  const daysLeft = Math.round((expDay - today) / 86400000)

  let status
  if (daysLeft < 0) status = EXPIRY_STATUS.EXPIRED
  else if (daysLeft <= withinDays) status = EXPIRY_STATUS.EXPIRING
  else status = EXPIRY_STATUS.VALID

  return { hasDate: true, issueDate: issue, expiryDate: expDay, daysLeft, status }
}

// วันที่แบบสั้นอ่านง่าย: 23 ก.ค. 2026
const TH_MONTHS = ['ม.ค.', 'ก.พ.', 'มี.ค.', 'เม.ย.', 'พ.ค.', 'มิ.ย.', 'ก.ค.', 'ส.ค.', 'ก.ย.', 'ต.ค.', 'พ.ย.', 'ธ.ค.']
export function formatThaiDate(d) {
  if (!d) return '—'
  const date = d instanceof Date ? d : new Date(d)
  if (Number.isNaN(date.getTime())) return '—'
  return `${date.getDate()} ${TH_MONTHS[date.getMonth()]} ${date.getFullYear()}`
}

// ข้อความวันคงเหลือแบบสั้น: "เหลือ 12 วัน" / "หมดอายุแล้ว 5 วัน" / "หมดอายุวันนี้"
export function daysLeftLabel(daysLeft) {
  if (daysLeft == null) return 'ยังไม่ระบุวันที่'
  if (daysLeft < 0) return `เลยกำหนด ${Math.abs(daysLeft)} วัน`
  if (daysLeft === 0) return 'หมดอายุวันนี้'
  return `เหลือ ${daysLeft} วัน`
}

// ป้ายไทยของแต่ละสถานะ — ใช้ซ้ำได้ทั้งตารางและ panel กระดิ่ง
export const STATUS_LABEL = {
  [EXPIRY_STATUS.EXPIRED]: 'หมดอายุแล้ว',
  [EXPIRY_STATUS.EXPIRING]: 'ใกล้หมดอายุ',
  [EXPIRY_STATUS.VALID]: 'ปกติ',
  [EXPIRY_STATUS.NO_DATE]: 'ยังไม่ระบุวันที่',
}
