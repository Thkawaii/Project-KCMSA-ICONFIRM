import { EXPIRY_STATUS, formatThaiDate } from './licenseExpiry.js';

// อายุใบอนุญาตนำออก = วันที่นำออกใบอนุญาต + 1 เดือน
export const EXPORT_LICENSE_VALIDITY_MONTHS = 1;

// Lead time — ต้องยื่นเรื่องให้ กสทช. ก่อนใบอนุญาตนำออกหมดอายุอย่างน้อย 15 วัน
export const EXPORT_LICENSE_LEAD_DAYS = 15;

export const LEAD_STATUS = {
  OVERDUE: 'LEAD_OVERDUE',
  DUE: 'LEAD_DUE',
  OK: 'LEAD_OK',
  NO_DATE: 'LEAD_NO_DATE'
};

export const LEAD_STATUS_LABEL = {
  [LEAD_STATUS.OVERDUE]: 'เลยกำหนดยื่น',
  [LEAD_STATUS.DUE]: 'ใกล้ถึงกำหนดยื่น',
  [LEAD_STATUS.OK]: 'ยังไม่ถึงกำหนดยื่น',
  [LEAD_STATUS.NO_DATE]: 'ยังไม่ระบุวันที่'
};

export const LEAD_BADGE_CLASS = {
  [LEAD_STATUS.OVERDUE]: 'il-badge il-badge-bad',
  [LEAD_STATUS.DUE]: 'il-badge il-badge-warn',
  [LEAD_STATUS.OK]: 'il-badge il-badge-ok',
  [LEAD_STATUS.NO_DATE]: 'il-badge il-badge-muted'
};

function atMidnight(d) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function toDate(raw) {
  if (!raw) return null;
  const d = raw instanceof Date ? new Date(raw.getTime()) : new Date(raw);
  return Number.isNaN(d.getTime()) ? null : d;
}

// บวกเดือนแบบไม่ล้นเดือน — 31 ม.ค. + 1 เดือน = 28/29 ก.พ. (ไม่ใช่ 2/3 มี.ค.)
export function addMonthsClamped(date, months) {
  const base = toDate(date);
  if (!base) return null;
  const y = base.getFullYear();
  const m = base.getMonth();
  const d = base.getDate();
  const lastDayOfTarget = new Date(y, m + months + 1, 0).getDate();
  return new Date(y, m + months, Math.min(d, lastDayOfTarget));
}

export function addDays(date, days) {
  const base = toDate(date);
  if (!base) return null;
  return new Date(base.getFullYear(), base.getMonth(), base.getDate() + days);
}

// วันหมดอายุที่ระบบใช้จริง
// ยึด "วันที่นำออกใบอนุญาต + 1 เดือน" เสมอ เพื่อกันไฟล์ Excel ที่ใส่วันหมดอายุมาไม่ตรงกติกา
// (เช่น ไฟล์ใส่ 31 ธ.ค. 2026 ทั้งที่นำออก 10 มี.ค. 2026 → ต้องเป็น 10 เม.ย. 2026)
export function exportExpiryDate(row) {
  if (!row) return null;
  const issue = toDate(row.IssueDate);
  if (issue) return addMonthsClamped(issue, EXPORT_LICENSE_VALIDITY_MONTHS);
  return toDate(row.ExpireDate);
}

// วันสุดท้ายที่ต้องยื่นเรื่องให้ กสทช. = วันหมดอายุ - 15 วัน
export function exportLeadTimeDate(row) {
  const expiry = exportExpiryDate(row);
  if (!expiry) return null;
  return addDays(expiry, -EXPORT_LICENSE_LEAD_DAYS);
}

const EMPTY = {
  hasDate: false,
  issueDate: null,
  expiryDate: null,
  daysLeft: null,
  status: EXPIRY_STATUS.NO_DATE,
  leadDate: null,
  leadDaysLeft: null,
  leadStatus: LEAD_STATUS.NO_DATE
};

// คำนวณวันหมดอายุ + Lead time ของใบอนุญาตนำออกในครั้งเดียว
export function computeExportLicenseDates(row, { withinDays = 7, leadWithinDays = 7 } = {}) {
  const expiry = exportExpiryDate(row);
  if (!expiry) return { ...EMPTY };

  const today = atMidnight(new Date());
  const expDay = atMidnight(expiry);
  const leadDay = addDays(expDay, -EXPORT_LICENSE_LEAD_DAYS);

  const daysLeft = Math.round((expDay - today) / 86400000);
  const leadDaysLeft = Math.round((leadDay - today) / 86400000);

  let status;
  if (daysLeft < 0) status = EXPIRY_STATUS.EXPIRED;
  else if (daysLeft <= withinDays) status = EXPIRY_STATUS.EXPIRING;
  else status = EXPIRY_STATUS.VALID;

  let leadStatus;
  if (leadDaysLeft < 0) leadStatus = LEAD_STATUS.OVERDUE;
  else if (leadDaysLeft <= leadWithinDays) leadStatus = LEAD_STATUS.DUE;
  else leadStatus = LEAD_STATUS.OK;

  return {
    hasDate: true,
    issueDate: toDate(row?.IssueDate),
    expiryDate: expDay,
    daysLeft,
    status,
    leadDate: leadDay,
    leadDaysLeft,
    leadStatus
  };
}

export function leadDaysLabel(leadDaysLeft) {
  if (leadDaysLeft == null) return 'ยังไม่ระบุวันที่';
  if (leadDaysLeft < 0) return `เลยกำหนดยื่นมา ${Math.abs(leadDaysLeft)} วัน`;
  if (leadDaysLeft === 0) return 'ต้องยื่นวันนี้';
  return `ต้องยื่นภายใน ${leadDaysLeft} วัน`;
}

export function leadTimeText(row) {
  const d = exportLeadTimeDate(row);
  return d ? formatThaiDate(d) : '—';
}
