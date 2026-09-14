import { EXPIRY_STATUS, formatThaiDate } from './licenseExpiry.js';

export const EXPORT_LICENSE_VALIDITY_MONTHS = 1;

export const EXPORT_LICENSE_LEAD_DAYS = 15;

export const EXPORT_LICENSE_LEAD_WARN_DAYS = 7;

export const LEAD_FILTER_DUE_SOON = 'LEAD_DUE_SOON';

export const LEAD_STATUS = {
  OVERDUE: 'LEAD_OVERDUE',
  DUE: 'LEAD_DUE',
  NO_DATE: 'LEAD_NO_DATE'
};

export const LEAD_STATUS_LABEL = {
  [LEAD_STATUS.OVERDUE]: 'เลยกำหนดยื่น',
  [LEAD_STATUS.DUE]: 'ถึงกำหนดยื่น',
  [LEAD_STATUS.NO_DATE]: 'ยังไม่ระบุวันที่'
};

export const LEAD_BADGE_CLASS = {
  [LEAD_STATUS.OVERDUE]: 'il-badge il-badge-bad',
  [LEAD_STATUS.DUE]: 'il-badge il-badge-ok',
  [LEAD_STATUS.NO_DATE]: 'il-badge il-badge-muted'
};

export function leadBadgeClass(info) {
  if (!info || !info.hasDate) return LEAD_BADGE_CLASS[LEAD_STATUS.NO_DATE];
  if (info.leadStatus === LEAD_STATUS.OVERDUE) return LEAD_BADGE_CLASS[LEAD_STATUS.OVERDUE];
  return LEAD_BADGE_CLASS[LEAD_STATUS.DUE];
}

function atMidnight(d) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function toDate(raw) {
  if (!raw) return null;
  const d = raw instanceof Date ? new Date(raw.getTime()) : new Date(raw);
  return Number.isNaN(d.getTime()) ? null : d;
}

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

export function exportExpiryDate(row) {
  if (!row) return null;
  const issue = toDate(row.IssueDate);
  if (issue) return addMonthsClamped(issue, EXPORT_LICENSE_VALIDITY_MONTHS);
  return toDate(row.ExpireDate);
}

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
  leadStatus: LEAD_STATUS.NO_DATE,
  leadUrgent: false,
  leadAlert: false
};

export function computeExportLicenseDates(row, { withinDays = 7, leadWarnDays = EXPORT_LICENSE_LEAD_WARN_DAYS } = {}) {
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

  const leadStatus = leadDaysLeft < 0 ? LEAD_STATUS.OVERDUE : LEAD_STATUS.DUE;
  const leadUrgent = leadStatus === LEAD_STATUS.DUE && leadDaysLeft <= leadWarnDays;

  return {
    hasDate: true,
    issueDate: toDate(row?.IssueDate),
    expiryDate: expDay,
    daysLeft,
    status,
    leadDate: leadDay,
    leadDaysLeft,
    leadStatus,
    leadUrgent,
    leadAlert: leadStatus === LEAD_STATUS.OVERDUE || leadUrgent
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
