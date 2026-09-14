import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { getImportLicenseAlerts } from '../api/importLicense.js';
import { getExportLicenseAlerts } from '../api/exportLicense.js';
import { useAppNavigate } from '../lib/nav.jsx';
import { formatThaiDate } from '../lib/licenseExpiry.js';
import {
  LEAD_STATUS,
  EXPORT_LICENSE_LEAD_DAYS,
  EXPORT_LICENSE_VALIDITY_MONTHS
} from '../lib/exportLicenseRules.js';
import { wasShownThisWeek, markShownThisWeek } from '../lib/weeklyPopupDismiss.js';
import { dismissKey, readDismissed } from '../lib/licenseDismiss.js';
import { exportDismissKey, readExportDismissed } from '../lib/exportLicenseDismiss.js';
import {
  ShieldCheckIcon,
  XMarkIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  ChevronRightIcon,
  ClockIcon,
  CheckCircleIcon
} from './icons.jsx';
import excavatorArt from '../assets/brand/kobelco-excavator.png';
import './LicenseWeeklyPopup.css';

const CLOSE_MS = 320;

const GROUP = {
  EXPIRED: 'EXPIRED',
  EXPIRING: 'EXPIRING',
  LEAD: 'LEAD'
};

const GROUP_LABEL = {
  [GROUP.EXPIRED]: 'หมดอายุแล้ว',
  [GROUP.EXPIRING]: 'ใกล้หมดอายุ',
  [GROUP.LEAD]: 'ต้องยื่น กสทช.'
};

const isExpiryAlert = it => it.Status === 'EXPIRED' || it.Status === 'EXPIRING';

const isLeadAlert = it => it.LeadStatus === LEAD_STATUS.OVERDUE || it.LeadUrgent === true;

const isLeadOnly = it => it.kind === 'export' && !isExpiryAlert(it) && isLeadAlert(it);

const isWeeklyAlert = it => isExpiryAlert(it) || isLeadAlert(it);

function groupOf(it) {
  if (it.Status === 'EXPIRED') return GROUP.EXPIRED;
  if (it.Status === 'EXPIRING') return GROUP.EXPIRING;
  return GROUP.LEAD;
}

function urgencyDays(it) {
  return isLeadOnly(it) ? it.LeadDaysLeft ?? 0 : it.DaysLeft ?? 0;
}

function shortDaysLabel(daysLeft) {
  if (daysLeft == null) return 'ไม่ระบุวันที่';
  if (daysLeft < 0) return `เลย ${Math.abs(daysLeft)} วัน`;
  if (daysLeft === 0) return 'วันนี้';
  return `เหลือ ${daysLeft} วัน`;
}

function shortLeadLabel(leadDaysLeft) {
  if (leadDaysLeft == null) return 'ไม่ระบุวันที่';
  if (leadDaysLeft < 0) return `เลยกำหนด ${Math.abs(leadDaysLeft)} วัน`;
  if (leadDaysLeft === 0) return 'ยื่นภายในวันนี้';
  return `ยื่นภายใน ${leadDaysLeft} วัน`;
}

function shortThaiDate(d) {
  const text = formatThaiDate(d);
  if (text === '—') return text;
  const year = new Date(d).getFullYear();
  return year === new Date().getFullYear() ? text.replace(` ${year}`, '') : text;
}

function weekRangeText() {
  const now = new Date();
  const mon = new Date(now.getFullYear(), now.getMonth(), now.getDate() - ((now.getDay() + 6) % 7));
  const sun = new Date(mon.getFullYear(), mon.getMonth(), mon.getDate() + 6);
  return `${shortThaiDate(mon)} – ${shortThaiDate(sun)}`;
}

export default function LicenseWeeklyPopup() {
  const navigate = useAppNavigate();
  const [items, setItems] = useState([]);
  const [open, setOpen] = useState(false);
  const [closing, setClosing] = useState(false);
  const [filter, setFilter] = useState('all');
  const [hasOverflow, setHasOverflow] = useState(false);
  const [shownTotal, setShownTotal] = useState(0);
  const scrollRef = useRef(null);

  useEffect(() => {
    let alive = true;
    async function load() {
      if (wasShownThisWeek()) return;
      const [imp, exp] = await Promise.allSettled([
        getImportLicenseAlerts({ onlyAlert: true }),
        getExportLicenseAlerts({ onlyAlert: true })
      ]);
      if (!alive) return;

      const impHidden = readDismissed();
      const expHidden = readExportDismissed();
      const isHidden = (map, key) => Object.prototype.hasOwnProperty.call(map, key);

      const impList = (imp.status === 'fulfilled' ? imp.value?.items || [] : [])
        .filter(it => !isHidden(impHidden, dismissKey(it)))
        .map(it => ({ ...it, kind: 'import' }))
        .filter(isExpiryAlert);

      const expList = (exp.status === 'fulfilled' ? exp.value?.items || [] : [])
        .filter(it => !isHidden(expHidden, exportDismissKey(it)))
        .map(it => ({ ...it, kind: 'export' }))
        .filter(isWeeklyAlert);

      const merged = [...impList, ...expList];
      if (merged.length === 0) return;
      if (wasShownThisWeek()) return;
      markShownThisWeek();
      setItems(merged);
      setOpen(true);
    }
    load();
    return () => {
      alive = false;
    };
  }, []);

  const handleClose = useCallback(() => {
    setClosing(true);
    const id = setTimeout(() => {
      setOpen(false);
      setClosing(false);
    }, CLOSE_MS);
    return () => clearTimeout(id);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onKey = e => {
      if (e.key === 'Escape') handleClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, handleClose]);

  const { counts, total, sections } = useMemo(() => {
    const byGroup = {
      [GROUP.EXPIRED]: [],
      [GROUP.EXPIRING]: [],
      [GROUP.LEAD]: []
    };
    items.forEach(it => byGroup[groupOf(it)].push(it));
    Object.values(byGroup).forEach(list =>
      list.sort((a, b) => urgencyDays(a) - urgencyDays(b))
    );

    const order = [GROUP.EXPIRED, GROUP.EXPIRING, GROUP.LEAD];
    return {
      counts: {
        [GROUP.EXPIRED]: byGroup[GROUP.EXPIRED].length,
        [GROUP.EXPIRING]: byGroup[GROUP.EXPIRING].length,
        [GROUP.LEAD]: byGroup[GROUP.LEAD].length
      },
      total: items.length,
      sections: order
        .filter(key => (filter === 'all' || filter === key) && byGroup[key].length > 0)
        .map(key => ({ key, rows: byGroup[key] }))
    };
  }, [items, filter]);

  useEffect(() => {
    if (!open || total === 0) return;
    const reduced =
      typeof window !== 'undefined' &&
      window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    if (reduced || typeof requestAnimationFrame !== 'function') {
      setShownTotal(total);
      return;
    }
    let raf = 0;
    const start = Date.now();
    const tick = () => {
      const p = Math.min((Date.now() - start) / 700, 1);
      const eased = 1 - Math.pow(1 - p, 3);
      setShownTotal(Math.round(total * eased));
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [open, total]);

  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const check = () => setHasOverflow(el.scrollHeight - el.clientHeight > 4);
    check();
    const onScroll = () => {
      const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 8;
      setHasOverflow(!atBottom && el.scrollHeight - el.clientHeight > 4);
    };
    el.addEventListener('scroll', onScroll);
    return () => el.removeEventListener('scroll', onScroll);
  }, [sections, open]);

  if (!open) return null;

  const titleOf = it =>
    it.kind === 'import'
      ? it.LicenseNo || 'ไม่มีเลขใบอนุญาต'
      : it.ExceptionLicense || 'ไม่มีเลขใบอนุญาต';

  const metaOf = it =>
    it.kind === 'import'
      ? `นำเข้า · ${it.InvoiceNo || '—'}`
      : `ส่งออก · ${it.Total || 1} เครื่อง`;

  const openItem = it => {
    handleClose();
    if (it.kind === 'import') {
      navigate('/warehouse', {
        focusLicense: it.LicenseNo || '',
        focusInvoice: it.InvoiceNo || '',
        focusTs: Date.now()
      });
    } else {
      navigate('/warehouse/export-license', {
        focusException: it.ExceptionLicense || '',
        focusTs: Date.now()
      });
    }
  };

  const goManage = () => {
    handleClose();
    navigate('/warehouse');
  };

  const pct = n => (total > 0 ? `${(n / total) * 100}%` : '0%');

  const chip = key => {
    const active = filter === key;
    const dim = filter !== 'all' && !active;
    return (
      <button
        type="button"
        className={
          `lwp-chip lwp-chip-${key.toLowerCase()}` +
          (active ? ' is-active' : '') +
          (dim ? ' is-dim' : '')
        }
        aria-pressed={active}
        onClick={() => setFilter(active ? 'all' : key)}
        title={active ? 'กดอีกครั้งเพื่อดูทั้งหมด' : `ดูเฉพาะ${GROUP_LABEL[key]}`}
      >
        <span className="lwp-chip-num">{counts[key]}</span>
        <span className="lwp-chip-lbl">{GROUP_LABEL[key]}</span>
      </button>
    );
  };

  return (
    <div
      className={'lwp-overlay' + (closing ? ' is-closing' : '')}
      role="presentation"
      onClick={handleClose}
    >
      <div
        className={'lwp-card' + (closing ? ' is-closing' : '')}
        role="dialog"
        aria-modal="true"
        aria-label={`แจ้งเตือนประจำสัปดาห์ · ${total} ใบอนุญาตต้องดำเนินการ`}
        onClick={e => e.stopPropagation()}
      >
        <div className="lwp-head">
          <img className="lwp-rig" src={excavatorArt} alt="" aria-hidden="true" />

          <button className="lwp-close" onClick={handleClose} aria-label="ปิด">
            <XMarkIcon className="size-4" />
          </button>

          <div className="lwp-head-body">
            <p className="lwp-eyebrow">
              <ShieldCheckIcon className="size-3" />
              แจ้งเตือนประจำสัปดาห์
            </p>

            <div className="lwp-count">
              <span className="lwp-count-num">{shownTotal}</span>
              <span className="lwp-count-lbl">
                ใบอนุญาต
                <br />
                ต้องดำเนินการ
              </span>
            </div>

            <p className="lwp-week">{weekRangeText()}</p>
          </div>
        </div>

        <div className="lwp-filters">
          {chip(GROUP.EXPIRED)}
          {chip(GROUP.EXPIRING)}
          {chip(GROUP.LEAD)}
        </div>

        <div className={'lwp-scroll' + (hasOverflow ? ' has-fade' : '')} ref={scrollRef}>
          {sections.length === 0 && (
            <div className="lwp-empty">
              <CheckCircleIcon className="size-5" />
              ไม่มีรายการในกลุ่มนี้
            </div>
          )}

          {sections.map((section, sIdx) => (
            <div className="lwp-section" key={section.key}>
              <div className={'lwp-group lwp-group-' + section.key.toLowerCase()}>
                <span className="lwp-group-dot" />
                {GROUP_LABEL[section.key]}
                <span className="lwp-group-count">{section.rows.length}</span>
              </div>

              {section.rows.map((it, idx) => {
                const leadOnly = isLeadOnly(it);
                const isExp = it.Status === 'EXPIRED';
                const leadLate = it.LeadStatus === LEAD_STATUS.OVERDUE;
                const iconKind = leadOnly ? 'lead' : it.kind;
                const daysCls = leadOnly
                  ? leadLate
                    ? 'is-lead-late'
                    : 'is-lead'
                  : isExp
                    ? 'is-expired'
                    : 'is-expiring';

                return (
                  <button
                    key={`${it.kind}-${titleOf(it)}-${idx}`}
                    type="button"
                    className={'lwp-item' + (leadOnly ? ' is-lead-row' : '')}
                    style={{ animationDelay: `${Math.min(sIdx * 2 + idx, 9) * 45}ms` }}
                    onClick={() => openItem(it)}
                    title={leadOnly ? 'ดูใบอนุญาตที่ต้องยื่น กสทช.' : 'ดูในหน้าจัดการใบอนุญาต'}
                  >
                    <span className={'lwp-item-icon lwp-item-icon-' + iconKind}>
                      {leadOnly ? (
                        <ClockIcon className="size-4" />
                      ) : it.kind === 'import' ? (
                        <ArrowDownTrayIcon className="size-4" />
                      ) : (
                        <ArrowUpTrayIcon className="size-4" />
                      )}
                    </span>

                    <span className="lwp-item-main">
                      <span className="lwp-item-top">
                        <span className="lwp-item-title">{titleOf(it)}</span>
                        <span className={'lwp-item-days ' + daysCls}>
                          {leadOnly ? shortLeadLabel(it.LeadDaysLeft) : shortDaysLabel(it.DaysLeft)}
                        </span>
                      </span>

                      <span className="lwp-item-sub">
                        <span className="lwp-item-meta">
                          {metaOf(it)} ·{' '}
                          {leadOnly
                            ? `กำหนดยื่น ${shortThaiDate(it.LeadTimeDate)}`
                            : `หมดอายุ ${shortThaiDate(it.ExpiryDate)}`}
                        </span>

                        {!leadOnly && it.kind === 'export' && isLeadAlert(it) && (
                          <span className={'lwp-item-lead' + (leadLate ? ' is-late' : '')}>
                            <ClockIcon className="size-3" />
                            {shortLeadLabel(it.LeadDaysLeft)}
                          </span>
                        )}
                      </span>
                    </span>
                  </button>
                );
              })}
            </div>
          ))}
        </div>

        <div className="lwp-foot">
          <button type="button" className="lwp-cta" onClick={goManage}>
            <span>ไปจัดการใบอนุญาต</span>
            <ChevronRightIcon className="size-4" />
          </button>
          <p className="lwp-rule">
            ใบนำออกอายุ {EXPORT_LICENSE_VALIDITY_MONTHS} เดือน · ต้องยื่น กสทช. ก่อนหมดอายุ{' '}
            {EXPORT_LICENSE_LEAD_DAYS} วัน
          </p>
        </div>
      </div>
    </div>
  );
}
