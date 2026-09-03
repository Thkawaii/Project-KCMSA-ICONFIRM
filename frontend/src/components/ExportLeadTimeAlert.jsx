import { useMemo, useState } from 'react';
import { computeExportLicenseDates, leadDaysLabel, LEAD_STATUS, EXPORT_LICENSE_LEAD_DAYS, EXPORT_LICENSE_LEAD_WARN_DAYS } from '../lib/exportLicenseRules.js';
import { formatThaiDate } from '../lib/licenseExpiry.js';
import { ExclamationTriangleIcon, ClockIcon, CheckCircleIcon, ChevronRightIcon, XMarkIcon } from './icons.jsx';

// ค่าฟิลเตอร์พิเศษ: "ถึงกำหนดยื่น" ที่เหลือเวลาไม่เกิน 7 วัน
export const LEAD_FILTER_DUE_SOON = 'LEAD_DUE_SOON';

const SNOOZE_KEY = 'iconfirm_export_lead_alert_snooze';
const MAX_LIST = 4;

function todayKey() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
function readSnoozed() {
  try {
    return localStorage.getItem(SNOOZE_KEY) === todayKey();
  } catch {
    return false;
  }
}

/**
 * แถบแจ้งเตือน Lead time ของหน้า Export License
 * วางไว้เหนือตาราง — เตือนว่ามีใบไหนต้องรีบยื่นเรื่องให้ กสทช. บ้าง
 * และกดที่ตัวเลขเพื่อกรองตารางด้านล่างได้ทันที
 */
export default function ExportLeadTimeAlert({ rows = [], activeFilter = 'all', onFilter, loading = false }) {
  const [snoozed, setSnoozed] = useState(readSnoozed);

  const { overdue, dueSoon, total } = useMemo(() => {
    // รวมเป็นรายใบอนุญาต ไม่ใช่รายเครื่อง — ใบเดียวมีหลายเครื่องจะได้ไม่ซ้ำเป็นสิบบรรทัด
    const groups = new Map();
    (rows || []).forEach(r => {
      const info = computeExportLicenseDates(r);
      if (!info.hasDate || !info.leadAlert) return;
      const key = String(r.ExportLicenseNo || r.ExceptionLicense || '').trim() || '(ไม่มีเลขใบอนุญาต)';
      const g = groups.get(key);
      if (!g) {
        groups.set(key, { key, info, count: 1, sample: r });
        return;
      }
      g.count += 1;
      if (info.leadDaysLeft < g.info.leadDaysLeft) {
        g.info = info;
        g.sample = r;
      }
    });

    const all = Array.from(groups.values()).sort((a, b) => a.info.leadDaysLeft - b.info.leadDaysLeft);
    const late = all.filter(g => g.info.leadStatus === LEAD_STATUS.OVERDUE);
    const soon = all.filter(g => g.info.leadStatus !== LEAD_STATUS.OVERDUE);
    return { overdue: late, dueSoon: soon, total: all.length };
  }, [rows]);

  if (loading || (rows || []).length === 0) return null;

  if (total === 0) {
    return (
      <div className="elt-alert elt-alert-ok" role="status">
        <span className="elt-icon">
          <CheckCircleIcon className="size-5" />
        </span>
        <div className="elt-body">
          <strong className="elt-title">Lead time ยังอยู่ในกำหนดทุกใบ</strong>
          <span className="elt-sub">
            ไม่มีใบอนุญาตส่งออกที่เลยกำหนดยื่น หรือใกล้ครบกำหนดยื่น กสทช. (ภายใน {EXPORT_LICENSE_LEAD_WARN_DAYS} วัน)
          </span>
        </div>
      </div>
    );
  }

  const tone = overdue.length > 0 ? 'danger' : 'warn';

  if (snoozed) {
    return (
      <button
        type="button"
        className={`elt-alert elt-alert-mini elt-alert-${tone}`}
        onClick={() => setSnoozed(false)}
        title="แสดงรายละเอียดการแจ้งเตือน Lead time"
      >
        <span className="elt-icon elt-icon-sm">
          {overdue.length > 0 ? <ExclamationTriangleIcon className="size-4" /> : <ClockIcon className="size-4" />}
        </span>
        <span className="elt-mini-text">
          Lead time ต้องดำเนินการ <strong>{total}</strong> ใบ
        </span>
        <span className="elt-mini-more">
          ดูรายละเอียด <ChevronRightIcon className="size-3" />
        </span>
      </button>
    );
  }

  const chip = (value, label, count, cls) => {
    if (count === 0) return null;
    const active = activeFilter === value;
    return (
      <button
        type="button"
        className={`elt-chip ${cls}` + (active ? ' is-active' : '')}
        onClick={() => onFilter?.(active ? 'all' : value)}
        title={active ? 'กดอีกครั้งเพื่อล้างตัวกรอง' : 'กรองตารางด้านล่างเฉพาะรายการนี้'}
      >
        <span className="elt-chip-num">{count}</span>
        <span className="elt-chip-lbl">{label}</span>
      </button>
    );
  };

  const shown = [...overdue, ...dueSoon].slice(0, MAX_LIST);

  return (
    <div className={`elt-alert elt-alert-${tone}`} role="alert">
      <span className="elt-icon">
        {overdue.length > 0 ? <ExclamationTriangleIcon className="size-5" /> : <ClockIcon className="size-5" />}
      </span>

      <div className="elt-body">
        <div className="elt-head">
          <div className="elt-head-text">
            <span className="elt-sub">
              ใบอนุญาตนำออกมีอายุ 1 เดือน · ต้องยื่นก่อนหมดอายุอย่างน้อย {EXPORT_LICENSE_LEAD_DAYS} วัน
            </span>
          </div>

          <div className="elt-chips">
            {chip(LEAD_STATUS.OVERDUE, 'เลยกำหนดยื่น', overdue.length, 'elt-chip-late')}
            {chip(LEAD_FILTER_DUE_SOON, `ใกล้ครบกำหนด ≤ ${EXPORT_LICENSE_LEAD_WARN_DAYS} วัน`, dueSoon.length, 'elt-chip-soon')}
          </div>

          <button
            type="button"
            className="elt-snooze"
            onClick={() => {
              try {
                localStorage.setItem(SNOOZE_KEY, todayKey());
              } catch {}
              setSnoozed(true);
            }}
            aria-label="ย่อการแจ้งเตือนไว้จนถึงพรุ่งนี้"
            title="ย่อไว้ก่อน (กลับมาเตือนใหม่พรุ่งนี้)"
          >
            <XMarkIcon className="size-4" />
          </button>
        </div>

        <div className="elt-list">
          {shown.map(g => {
            const late = g.info.leadStatus === LEAD_STATUS.OVERDUE;
            return (
              <button
                key={g.key}
                type="button"
                className="elt-item"
                onClick={() => onFilter?.(late ? LEAD_STATUS.OVERDUE : LEAD_FILTER_DUE_SOON, g.key)}
                title="กรองตารางด้านล่างไปที่ใบอนุญาตนี้"
              >
                <span className={`elt-dot ${late ? 'is-late' : 'is-soon'}`} />
                <span className="elt-item-license il-mono">{g.key}</span>
                <span className="elt-item-meta">
                  {g.count} เครื่อง
                  {g.sample?.IssueDate ? ` · นำออก ${formatThaiDate(g.sample.IssueDate)}` : ''}
                </span>
                <span className="elt-item-date">ยื่นภายใน {formatThaiDate(g.info.leadDate)}</span>
                <span className={`elt-item-days ${late ? 'is-late' : 'is-soon'}`}>
                  {leadDaysLabel(g.info.leadDaysLeft)}
                </span>
              </button>
            );
          })}

          {total > shown.length && (
            <button
              type="button"
              className="elt-more"
              onClick={() => onFilter?.(overdue.length > 0 ? LEAD_STATUS.OVERDUE : LEAD_FILTER_DUE_SOON)}
            >
              และอีก {total - shown.length} ใบ — กดเพื่อกรองดูในตารางด้านล่าง
              <ChevronRightIcon className="size-3" />
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
