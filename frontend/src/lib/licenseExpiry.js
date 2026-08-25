
export const LICENSE_VALIDITY_MONTHS = 6

export const EXPIRY_STATUS = {
  EXPIRED: 'EXPIRED',
  EXPIRING: 'EXPIRING',
  VALID: 'VALID',
  NO_DATE: 'NO_DATE',
}

function atMidnight(d) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

export function computeLicenseExpiry(issueDateRaw, withinDays = 30) {
  if (!issueDateRaw) {
    return { hasDate: false, issueDate: null, expiryDate: null, daysLeft: null, status: EXPIRY_STATUS.NO_DATE }
  }

  const issue = new Date(issueDateRaw)
  if (Number.isNaN(issue.getTime())) {
    return { hasDate: false, issueDate: null, expiryDate: null, daysLeft: null, status: EXPIRY_STATUS.NO_DATE }
  }

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

const TH_MONTHS = ['ม.ค.', 'ก.พ.', 'มี.ค.', 'เม.ย.', 'พ.ค.', 'มิ.ย.', 'ก.ค.', 'ส.ค.', 'ก.ย.', 'ต.ค.', 'พ.ย.', 'ธ.ค.']
export function formatThaiDate(d) {
  if (!d) return '—'
  const date = d instanceof Date ? d : new Date(d)
  if (Number.isNaN(date.getTime())) return '—'
  return `${date.getDate()} ${TH_MONTHS[date.getMonth()]} ${date.getFullYear()}`
}

export function daysLeftLabel(daysLeft) {
  if (daysLeft == null) return 'ยังไม่ระบุวันที่'
  if (daysLeft < 0) return `เลยกำหนด ${Math.abs(daysLeft)} วัน`
  if (daysLeft === 0) return 'หมดอายุวันนี้'
  return `เหลือ ${daysLeft} วัน`
}

export const STATUS_LABEL = {
  [EXPIRY_STATUS.EXPIRED]: 'หมดอายุแล้ว',
  [EXPIRY_STATUS.EXPIRING]: 'ใกล้หมดอายุ',
  [EXPIRY_STATUS.VALID]: 'ปกติ',
  [EXPIRY_STATUS.NO_DATE]: 'ยังไม่ระบุวันที่',
}
