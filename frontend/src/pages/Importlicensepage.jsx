import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getImportLicenseItems, getImportLicenseSummary, uploadImportLicense, previewImportLicense, deleteImportLicenseItem, bulkDeleteImportLicenseItems, clearImportLicense, renewImportLicense, setImportLicenseComplete, updateImportLicenseItem } from '../api/importLicense.js';
import { EditableCell, LockBadge, useLiveRefresh } from '../components/InlineEdit.jsx';
import { confirmUploadDeletes, syncResultParts } from '../lib/uploadSync.js';
import { getExportLicense, getExportLicenseTrace, uploadExportLicense, previewExportLicense, deleteExportLicense, bulkDeleteExportLicense, clearExportLicense, renewExportLicense, setExportLicenseComplete, updateExportLicense } from '../api/exportLicense.js';
import { PreviewResult, ChangePreview, ExtraColumnsCell } from '../components/FormatTools.jsx';
import AppShell from '../components/AppShell.jsx';
import FileDropZone from '../components/Filedropzone.jsx';
import SelectField from '../components/Selectfield.jsx';
import { confirmDelete, confirmComplete, toastError, toastSuccess, promptRenewDays } from '../lib/toast.js';
import { computeLicenseExpiry, formatThaiDate, daysLeftLabel, STATUS_LABEL, EXPIRY_STATUS, COMPLETED_FILTER, COMPLETED_LABEL, isLicenseCompleted } from '../lib/licenseExpiry.js';
import { computeExportLicenseDates, exportLeadTimeDate, leadDaysLabel, leadBadgeClass, LEAD_STATUS, LEAD_STATUS_LABEL, LEAD_BADGE_CLASS, EXPORT_LICENSE_LEAD_DAYS } from '../lib/exportLicenseRules.js';
import { useDailyTick } from '../lib/useDailyTick.js';
import { useAppParams } from '../lib/nav.jsx';
import { buildStyledXlsxWorkbookBlob, downloadBlob } from '../lib/xlsx.js';
import PeriodRangePicker from '../components/PeriodRangePicker.jsx';
import { inPeriod, periodRangeLabel, periodFileTag } from '../lib/dateRange.js';
import { ArrowPathIcon, ArrowsRightLeftIcon, CheckBadgeIcon, CheckBadgeSolidIcon, CheckCircleIcon, CheckIcon, ChevronDoubleLeftIcon, ChevronDoubleRightIcon, ChevronDownIcon, ChevronLeftIcon, ChevronRightIcon, ClipboardDocumentCheckIcon, ClockIcon, CubeIcon, DocumentTextIcon, MinusIcon, RectangleStackIcon, ReceiptPercentIcon, ShieldCheckIcon, Squares2X2Icon, TagIcon, TrashIcon, TruckIcon, WrenchScrewdriverIcon, XMarkIcon } from '../components/icons.jsx';
export const WH_NAV_ITEMS = [{
  to: '/warehouse',
  label: 'Import License',
  icon: <DocumentTextIcon className="size-4" />,
  roles: ['LOG']
}, {
  to: '/warehouse/export-license',
  label: 'Export License',
  icon: <ReceiptPercentIcon className="size-4" />,
  roles: ['LOG']
}, {
  to: '/warehouse/confirm',
  label: 'Part Confirmation',
  icon: <ClipboardDocumentCheckIcon className="size-4" />,
  roles: ['WH']
}];
const EXPIRY_BADGE_CLASS = {
  [EXPIRY_STATUS.EXPIRED]: 'il-badge il-badge-bad',
  [EXPIRY_STATUS.EXPIRING]: 'il-badge il-badge-warn',
  [EXPIRY_STATUS.VALID]: 'il-badge il-badge-ok',
  [EXPIRY_STATUS.NO_DATE]: 'il-badge il-badge-muted'
};

function StatLabel({
  parts
}) {
  return <span className="il-stat-label-text">
      {parts.map((part, i) => <Fragment key={i}>
          {i > 0 && <wbr />}
          <span className="il-nobr">{part}</span>
        </Fragment>)}
    </span>;
}
function CompleteBadge({
  row,
  date,
  dateLabel = 'หมดอายุ'
}) {
  const by = row?.CompletedBy || '';
  const at = row?.CompletedAt ? formatThaiDate(row.CompletedAt) : '';
  return <div className="il-expiry-cell il-complete-cell">
      <span className="il-badge il-badge-complete">
        <CheckBadgeIcon className="size-3.5" />
        {COMPLETED_LABEL}
      </span>
      {date && <span className="il-complete-date">
          {dateLabel} {formatThaiDate(date)}
        </span>}
      {(by || at) && <span className="il-complete-meta">
          {[by, at].filter(Boolean).join(' · ')}
        </span>}
    </div>;
}

function CompleteFlag({
  show
}) {
  if (!show) return null;
  return <span className="il-complete-flag" title={COMPLETED_LABEL} aria-label={COMPLETED_LABEL}>
      <CheckCircleIcon className="size-4" />
    </span>;
}

function licenseOptionLabel(licenseNo, total, done, extra = '') {
  const nb = text => text.replace(/ /g, '\u00A0');
  const parts = [licenseNo || '(ไม่มีเลขใบอนุญาต)'];
  if (extra) parts.push(nb(extra));
  parts.push(nb(`${total} เครื่อง`));
  if (done > 0) parts.push(nb(`เสร็จสิ้น ${done}/${total}`));
  return parts.join(' · ');
}

function CompletedOptionIcon() {
  return <span className="il-option-complete" role="img" title="เสร็จสิ้นแล้ว" aria-label="เสร็จสิ้นแล้ว">
      <CheckBadgeSolidIcon className="il-option-complete-icon" aria-hidden="true" />
    </span>;
}

const ALL_COUNTRIES = 'all';
const NO_COUNTRY = '__no_country__';
const NO_COUNTRY_LABEL = 'ไม่ระบุประเทศ';
function countryKey(value) {
  const key = String(value ?? '').trim().replace(/\s+/g, ' ').toUpperCase();
  return key || NO_COUNTRY;
}
function countryLabel(key) {
  if (key === NO_COUNTRY) return NO_COUNTRY_LABEL;
  return key.toLowerCase().replace(/(^|[\s-])(\S)/g, (_, sep, ch) => sep + ch.toUpperCase());
}
function buildCountryOptions(values) {
  const keys = new Set(values.map(countryKey));
  const list = Array.from(keys).filter(k => k !== NO_COUNTRY).sort((a, b) => a.localeCompare(b));
  if (keys.has(NO_COUNTRY)) list.push(NO_COUNTRY);
  return [{
    value: ALL_COUNTRIES,
    label: 'ทุกประเทศ'
  }, ...list.map(k => ({
    value: k,
    label: countryLabel(k)
  }))];
}

function LicenseRefSummary({
  label,
  values
}) {
  return <span className="il-ref-seg">
      {label} <span className="il-ref-first">{values[0] || '—'}</span>
      {values.length > 1 && <span className="il-ref-more">+{values.length - 1}</span>}
    </span>;
}
function LicenseRefList({
  label,
  values
}) {
  if (values.length === 0) return null;
  return <div className="il-ref-list">
      <span className="il-ref-list-label">
        {label} <span className="il-ref-list-count">{values.length}</span>
      </span>
      <div className="il-ref-chips">
        {values.map(v => <span key={v} className="il-ref-chip il-mono">
            {v}
          </span>)}
      </div>
    </div>;
}

function SelectCheckbox({
  checked,
  indeterminate = false,
  onChange,
  label,
  title,
  disabled = false
}) {
  // จำว่ากด Shift ค้างไว้ตอนคลิกหรือไม่ — ใช้เลือกเป็นช่วง (คลิกแถวแรก แล้ว Shift+คลิกแถวสุดท้าย)
  const shiftRef = useRef(false);
  return <label className={'il-check' + (disabled ? ' il-check-disabled' : '')} title={title} onClick={e => e.stopPropagation()} onMouseDown={e => {
    shiftRef.current = e.shiftKey;
    if (e.shiftKey) e.preventDefault(); // กันไม่ให้ข้อความในตารางถูกไฮไลต์ตอน Shift+คลิก
  }}>
      <input type="checkbox" checked={checked} disabled={disabled} ref={el => {
      if (el) el.indeterminate = indeterminate && !checked;
    }} onChange={e => {
      const shift = shiftRef.current || !!e.nativeEvent?.shiftKey;
      shiftRef.current = false;
      onChange(e.target.checked, shift);
    }} aria-label={label} />
      <span className="il-check-box" aria-hidden="true">
        {indeterminate && !checked ? <MinusIcon className="size-3" /> : <CheckIcon className="size-3" />}
      </span>
    </label>;
}

function useRowSelection(visibleRows) {
  const [selected, setSelected] = useState(() => new Set());
  const lastIdRef = useRef(null);
  useEffect(() => {
    setSelected(prev => {
      if (prev.size === 0) return prev;
      const alive = new Set(visibleRows.map(r => r.ID));
      let changed = false;
      const next = new Set();
      prev.forEach(id => {
        if (alive.has(id)) next.add(id);else changed = true;
      });
      return changed ? next : prev;
    });
  }, [visibleRows]);
  // shift=true: เลือก/ยกเลิกทุกแถวระหว่างแถวที่คลิกล่าสุดกับแถวนี้ (ข้ามหน้าได้ ตามลำดับในตาราง)
  const toggleOne = useCallback((id, shift = false) => {
    const lastId = lastIdRef.current;
    lastIdRef.current = id;
    setSelected(prev => {
      const next = new Set(prev);
      const turnOn = !prev.has(id);
      if (shift && lastId != null && lastId !== id) {
        const a = visibleRows.findIndex(r => r.ID === lastId);
        const b = visibleRows.findIndex(r => r.ID === id);
        if (a !== -1 && b !== -1) {
          const [from, to] = a < b ? [a, b] : [b, a];
          for (let i = from; i <= to; i++) {
            if (turnOn) next.add(visibleRows[i].ID);else next.delete(visibleRows[i].ID);
          }
          return next;
        }
      }
      if (turnOn) next.add(id);else next.delete(id);
      return next;
    });
  }, [visibleRows]);
  const setGroup = useCallback((ids, on) => {
    setSelected(prev => {
      const next = new Set(prev);
      ids.forEach(id => {
        if (on) next.add(id);else next.delete(id);
      });
      return next;
    });
  }, []);
  const replaceWith = useCallback(ids => {
    lastIdRef.current = null;
    setSelected(new Set(ids));
  }, []);
  const clear = useCallback(() => {
    lastIdRef.current = null;
    setSelected(new Set());
  }, []);
  const toggleAll = useCallback(on => {
    lastIdRef.current = null;
    setSelected(on ? new Set(visibleRows.map(r => r.ID)) : new Set());
  }, [visibleRows]);
  // กลับการเลือก: ที่เลือกอยู่ → ไม่เลือก, ที่ไม่ได้เลือก → เลือก
  const invert = useCallback(() => {
    lastIdRef.current = null;
    setSelected(prev => new Set(visibleRows.filter(r => !prev.has(r.ID)).map(r => r.ID)));
  }, [visibleRows]);
  const allSelected = visibleRows.length > 0 && selected.size >= visibleRows.length;
  const someSelected = selected.size > 0 && !allSelected;
  return {
    selected,
    toggleOne,
    setGroup,
    replaceWith,
    clear,
    toggleAll,
    invert,
    allSelected,
    someSelected
  };
}

const fmtCount = n => Number(n || 0).toLocaleString('en-US');

// LicenseCheckSelect: dropdown ใบอนุญาต ที่มีช่องติ๊กหน้าแต่ละใบ
// - คลิกชื่อใบ   → ดูเฉพาะใบนั้นในตาราง (เหมือนเดิม) แล้วปิด dropdown
// - ติ๊กช่องหน้าใบ → เลือกไว้หลายใบ (dropdown ไม่ปิด) แล้วใช้ปุ่มเสร็จสิ้น/ลบ ที่แถบด้านล่าง
// - ติ๊กช่อง "ทุกใบอนุญาต" → เลือก/ยกเลิกทุกใบ
function LicenseCheckSelect({
  value,
  onChange,
  options = [],
  allValue,
  checked,
  onToggleCheck,
  onCheckAll
}) {
  const [open, setOpen] = useState(false);
  const boxRef = useRef(null);
  useEffect(() => {
    if (!open) return undefined;
    function onOutside(e) {
      if (boxRef.current && !boxRef.current.contains(e.target)) setOpen(false);
    }
    function onKey(e) {
      if (e.key === 'Escape') setOpen(false);
    }
    document.addEventListener('mousedown', onOutside);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onOutside);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);
  const selected = options.find(o => o.value === value);
  const licenseOpts = options.filter(o => o.value !== allValue);
  const checkedCount = licenseOpts.filter(o => checked.has(o.value)).length;
  const allChecked = licenseOpts.length > 0 && checkedCount === licenseOpts.length;
  function pick(v) {
    onChange(v);
    setOpen(false);
  }
  return <div className="sf il-lcs" ref={boxRef}>
      <button type="button" className={'sf-trigger' + (open ? ' sf-trigger-open' : '')} onClick={() => setOpen(o => !o)} aria-haspopup="listbox" aria-expanded={open}>
        <span className="sf-value-wrap">
          <span className="sf-value">{selected ? selected.label : options[0]?.label}</span>
          {selected?.suffix && <span className="sf-suffix">{selected.suffix}</span>}
        </span>
        {checkedCount > 0 && <span className="il-lcs-badge">เลือกไว้ {fmtCount(checkedCount)} ใบ</span>}
        <span className="sf-chevron" aria-hidden="true">
          <svg width="12" height="8" viewBox="0 0 12 8" fill="none">
            <path d="M1 1.5L6 6.5L11 1.5" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </span>
      </button>

      {open && <ul className="sf-list il-lcs-list" role="listbox">
          {options.map(o => {
        const isAll = o.value === allValue;
        const isChecked = isAll ? allChecked : checked.has(o.value);
        return <li key={o.value || '__empty__'} className={'il-lcs-row' + (isAll ? ' il-lcs-row-all' : '') + (o.value === value ? ' sf-option-selected' : '') + (isChecked && !isAll ? ' il-lcs-row-checked' : '')}>
                <SelectCheckbox checked={isChecked} indeterminate={isAll && checkedCount > 0} onChange={on => isAll ? onCheckAll(on) : onToggleCheck(o.value)} label={isAll ? 'เลือกทุกใบอนุญาต' : `เลือก ${o.value}`} title={isAll ? allChecked ? 'ยกเลิกการเลือกทุกใบ' : 'เลือกทุกใบ' : 'เลือกได้หลายใบ'} />
                <button type="button" role="option" aria-selected={o.value === value} className="il-lcs-label" onClick={() => pick(o.value)}>
                  {o.label}
                  {o.suffix && <span className="sf-suffix sf-option-suffix">{o.suffix}</span>}
                </button>
              </li>;
      })}
          {licenseOpts.length === 0 && <li className="sf-empty">ไม่มีใบอนุญาต</li>}
        </ul>}
    </div>;
}

// แถบปุ่มสำหรับใบอนุญาตที่ติ๊กไว้ (แสดงเมื่อติ๊กอย่างน้อย 1 ใบ)
function LicenseActionBar({
  licenseCount,
  machineCount,
  openCount,
  doneCount,
  busy,
  onComplete,
  onUncomplete,
  onDelete,
  onClear
}) {
  if (licenseCount === 0) return null;
  return <div className="il-selection-bar il-license-action-bar" role="status">
      <div className="il-selection-info">
        <span className="il-selection-count">
          <CheckBadgeIcon className="size-4" />
          เลือกไว้ {fmtCount(licenseCount)} ใบอนุญาต ({fmtCount(machineCount)} เครื่อง)
        </span>
        <span className="il-selection-hint">
          {openCount > 0 ? `ยังไม่เสร็จสิ้น ${fmtCount(openCount)} เครื่อง` : 'เสร็จสิ้นแล้วทั้งหมด'}
          {doneCount > 0 && openCount > 0 ? ` · เสร็จสิ้นแล้ว ${fmtCount(doneCount)} เครื่อง` : ''}
        </span>
      </div>
      <div className="il-selection-actions">
        <button type="button" className="il-complete-btn" onClick={onComplete} disabled={busy || openCount === 0}>
          <CheckBadgeIcon className="size-4" />
          เสร็จสิ้น {fmtCount(licenseCount)} ใบ
        </button>
        <button type="button" className="il-uncomplete-btn" onClick={onUncomplete} disabled={busy || doneCount === 0}>
          <ArrowPathIcon className="size-4" />
          ยกเลิกเสร็จสิ้น
        </button>
        <button type="button" className="il-bulk-delete-btn" onClick={onDelete} disabled={busy}>
          <TrashIcon className="size-4" />
          ลบ {fmtCount(licenseCount)} ใบ
        </button>
        <button type="button" className="il-selection-clear" onClick={onClear} disabled={busy}>
          <XMarkIcon className="size-4" />
          ล้างการเลือก
        </button>
      </div>
    </div>;
}

// useCheckedSet: เก็บค่าที่ติ๊กไว้ และตัดค่าที่ไม่มีอยู่แล้วออกอัตโนมัติ (เช่น หลังลบใบ)
function useCheckedSet(validValues) {
  const [checked, setChecked] = useState(() => new Set());
  useEffect(() => {
    setChecked(prev => {
      if (prev.size === 0) return prev;
      const alive = new Set(validValues);
      const next = new Set([...prev].filter(v => alive.has(v)));
      return next.size === prev.size ? prev : next;
    });
  }, [validValues]);
  const toggle = useCallback(v => setChecked(prev => {
    const next = new Set(prev);
    if (next.has(v)) next.delete(v);else next.add(v);
    return next;
  }), []);
  const setAll = useCallback(on => setChecked(on ? new Set(validValues) : new Set()), [validValues]);
  const clear = useCallback(() => setChecked(new Set()), []);
  return {
    checked,
    toggle,
    setAll,
    clear
  };
}

function SelectionBar({
  selectedRows,
  onComplete,
  onUncomplete,
  onDelete,
  onInvert,
  onClear,
  busy,
  lockedCount = 0,
  allSelected = false,
  filterActive = false
}) {
  const count = selectedRows.length;
  if (count === 0) return null;
  const doneCount = selectedRows.filter(isLicenseCompleted).length;
  const openCount = count - doneCount;
  const deletableCount = count - lockedCount;
  return <div className="il-selection-bar" role="status">
      <div className="il-selection-info">
        <span className="il-selection-count">
          <CheckBadgeIcon className="size-4" />
          เลือกไว้ {fmtCount(count)} รายการ
          {allSelected && <span className="il-selection-scope">
              {filterActive ? 'ทุกรายการตามตัวกรอง' : 'ทุกรายการ'}
            </span>}
        </span>
        <span className="il-selection-hint">
          {openCount > 0 ? `ยังไม่เสร็จสิ้น ${fmtCount(openCount)} รายการ` : 'เสร็จสิ้นแล้วทั้งหมด'}
          {doneCount > 0 && openCount > 0 ? ` · เสร็จสิ้นแล้ว ${fmtCount(doneCount)} รายการ` : ''}
          {lockedCount > 0 ? ` · สแกนแล้ว ${fmtCount(lockedCount)} รายการ (ลบไม่ได้)` : ''}
        </span>
      </div>
      <div className="il-selection-actions">
        <button type="button" className="il-complete-btn" onClick={onComplete} disabled={busy || openCount === 0}>
          <CheckBadgeIcon className="size-4" />
          เสร็จสิ้น {fmtCount(openCount)} รายการ
        </button>
        <button type="button" className="il-uncomplete-btn" onClick={onUncomplete} disabled={busy || doneCount === 0}>
          <ArrowPathIcon className="size-4" />
          ยกเลิกสถานะ
        </button>
        {onDelete && <button type="button" className="il-bulk-delete-btn" onClick={onDelete} disabled={busy || deletableCount === 0} title={deletableCount === 0 ? 'รายการที่เลือกสแกนแล้วทั้งหมด ลบไม่ได้' : ''}>
            <TrashIcon className="size-4" />
            ลบ {fmtCount(deletableCount)} รายการ
          </button>}
        {onInvert && !allSelected && <button type="button" className="il-selection-clear" onClick={onInvert} disabled={busy} title="สลับ: รายการที่เลือกอยู่จะถูกยกเลิก และรายการที่เหลือจะถูกเลือกแทน">
            <ArrowsRightLeftIcon className="size-4" />
            กลับการเลือก
          </button>}
        <button type="button" className="il-selection-clear" onClick={onClear} disabled={busy}>
          <XMarkIcon className="size-4" />
          ล้างการเลือก
        </button>
      </div>
    </div>;
}

function ExpiryCell({
  row,
  issueDate,
  expireDate
}) {
  const exp = expireDate ? computeExpireStatus(expireDate, 30) : computeLicenseExpiry(issueDate);
  if (isLicenseCompleted(row)) return <CompleteBadge row={row} date={exp.expiryDate} />;
  return <div className="il-expiry-cell">
      <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
      {exp.hasDate && <>
          <span>{formatThaiDate(exp.expiryDate)}</span>
          <span className="il-expiry-days">{daysLeftLabel(exp.daysLeft)}</span>
        </>}
    </div>;
}
export default function ImportLicensePage() {
  const today = useDailyTick();
  const params = useAppParams();
  const [items, setItems] = useState([]);
  const [summary, setSummary] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [selectedLot, setSelectedLot] = useState('');
  const [search, setSearch] = useState('');
  const [countryFilter, setCountryFilter] = useState(ALL_COUNTRIES);
  const [expiryFilter, setExpiryFilter] = useState('all');
  const [pageSize, setPageSize] = useState(25);
  const [page, setPage] = useState(1);
  const [detailRow, setDetailRow] = useState(null);
  const [file, setFile] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [uploadMsg, setUploadMsg] = useState(null);
  const [previewData, setPreviewData] = useState(null);
  const [previewing, setPreviewing] = useState(false);
  const [completing, setCompleting] = useState(false);
  async function handlePreview() {
    if (!file) {
      setUploadMsg({
        error: 'กรุณาเลือกไฟล์ก่อนตรวจสอบ'
      });
      return;
    }
    setPreviewing(true);
    setPreviewData(null);
    try {
      const data = await previewImportLicense(file);
      setPreviewData(data);
    } catch (err) {
      setUploadMsg({
        error: err.message || 'ตรวจสอบไฟล์ไม่สำเร็จ'
      });
    } finally {
      setPreviewing(false);
    }
  }
  const loadSeq = useRef(0);
  async function loadAll(silent = false) {
    const seq = ++loadSeq.current;
    if (!silent) {
      setLoading(true);
      setLoadError('');
    }
    try {
      const [rows, sum] = await Promise.all([getImportLicenseItems(), getImportLicenseSummary()]);
      if (seq !== loadSeq.current) return;
      setItems(rows || []);
      setSummary(sum || []);
    } catch (err) {
      if (!silent && seq === loadSeq.current) setLoadError(err.message || 'โหลดบัญชีใบอนุญาตนำเข้าไม่สำเร็จ');
    } finally {
      if (!silent && seq === loadSeq.current) setLoading(false);
    }
  }
  useEffect(() => {
    loadAll();
  }, []);
  useLiveRefresh(() => loadAll(true));
  async function saveImportField(row, key, value) {
    try {
      const updated = await updateImportLicenseItem(row.ID, {
        [key]: value
      });
      setItems(list => list.map(r => r.ID === row.ID ? {
        ...r,
        ...updated
      } : r));
      if (['LicenseNo', 'InvoiceNo', 'DeclarationNo', 'Model'].includes(key)) {
        getImportLicenseSummary().then(sum => setSummary(sum || [])).catch(() => {});
      }
    } catch (err) {
      if (err?.status === 409) loadAll(true);
      throw err;
    }
  }
  useEffect(() => {
    setPage(1);
  }, [selectedLot, search, countryFilter, expiryFilter, pageSize]);
  useEffect(() => {
    const lic = (params?.focusLicense || '').trim();
    if (!lic) return;
    setCountryFilter(ALL_COUNTRIES);
    setExpiryFilter('all');
    setSelectedLot('');
    setSearch(lic);
  }, [params?.focusLicense, params?.focusInvoice, params?.focusTs]);
  useEffect(() => {
    const lic = (params?.focusLicense || '').trim();
    const inv = (params?.focusInvoice || '').trim();
    if (!lic || !inv) return;
    const key = `${lic}|${inv}`;
    if (summary.some(s => `${s.LicenseNo}|${s.InvoiceNo}` === key)) {
      setSelectedLot(key);
    }
  }, [summary, params?.focusLicense, params?.focusInvoice, params?.focusTs]);
  async function handleUpload() {
    if (!file) {
      setUploadMsg({
        error: 'กรุณาเลือกไฟล์ Excel หรือ CSV ก่อน'
      });
      return;
    }
    setUploading(true);
    setUploadMsg(null);
    try {
      const summary = previewData?.summary || (await previewImportLicense(file))?.summary;
      if (!(await confirmUploadDeletes(summary, file.name))) {
        return;
      }
      const result = await uploadImportLicense(file);
      setUploadMsg({
        success: [`เพิ่มใหม่ ${result.imported ?? 0}`, `อัปเดต ${result.updated ?? 0}`, ...(result.unchanged ? [`เหมือนเดิม ${result.unchanged}`] : []), ...(result.locked ? [`สแกนแล้ว ไม่อัปเดต ${result.locked}`] : []), ...syncResultParts(result), `ข้าม ${result.skipped ?? 0}`].join(' · '),
        problems: result.problems || []
      });
      setFile(null);
      setPreviewData(null);
      await loadAll();
    } catch (err) {
      setUploadMsg({
        error: err.message || 'อัปโหลดไม่สำเร็จ'
      });
    } finally {
      setUploading(false);
    }
  }
  async function handleDeleteRow(row) {
    const ok = await confirmDelete({
      text: `ลบหมายเลขเครื่อง ${row.MachineNo} ออกจากบัญชี?`
    });
    if (!ok) return;
    try {
      await deleteImportLicenseItem(row.ID);
      await loadAll();
      toastSuccess(`ลบ ${row.MachineNo} แล้ว`);
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
      if (err?.status === 409) loadAll(true);
    }
  }
  async function handleClearAllImport() {
    const ok = await confirmDelete({
      text: 'ลบใบอนุญาตนำเข้าทั้งหมดออกจากระบบ? กู้คืนไม่ได้',
      confirmText: 'ลบทั้งหมด'
    });
    if (!ok) return;
    try {
      const res = await clearImportLicense('', '', true);
      setSelectedLot('');
      await loadAll();
      toastSuccess(`ลบแล้ว ${res.deleted ?? 0} เครื่อง`);
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    }
  }
  async function handleClearLicense(lot) {
    const licenseNo = lot?.LicenseNo ?? '';
    const invoiceNo = lot?.InvoiceNo ?? '';
    const label = licenseNo || (invoiceNo ? `Invoice ${invoiceNo}` : 'ล็อตนี้ (ไม่มีเลขใบอนุญาต)');
    const ok = await confirmDelete({
      text: `ลบ ${label} ออกจากระบบทั้งล็อต? กู้คืนไม่ได้`,
      confirmText: 'ลบทั้งใบ'
    });
    if (!ok) return;
    try {
      await clearImportLicense(licenseNo, invoiceNo);
      setSelectedLot('');
      await loadAll();
      toastSuccess(`ลบ ${label} แล้ว`);
    } catch (err) {
      const msg = err.message || 'ลบไม่สำเร็จ';
      setLoadError(msg);
      toastError(msg);
    }
  }
  async function handleRenewLicense(lot) {
    const licenseNo = lot?.LicenseNo ?? '';
    const invoiceNo = lot?.InvoiceNo ?? '';
    const label = licenseNo || (invoiceNo ? `Invoice ${invoiceNo}` : 'ล็อตนี้ (ไม่มีเลขใบอนุญาต)');
    const lotRows = items.filter(r => (r.LicenseNo || '') === licenseNo && (r.InvoiceNo || '') === invoiceNo);
    const sample = lotRows.find(r => r.IssueDate) || lotRows[0];
    const curExp = sample ? computeLicenseExpiry(sample.IssueDate) : null;
    const curLine = curExp?.hasDate ? `<div class="scan-popup-hint">วันหมดอายุปัจจุบัน: <b>${formatThaiDate(curExp.expiryDate)}</b> (${daysLeftLabel(curExp.daysLeft)})</div>` : '';
    const days = await promptRenewDays({
      title: `ต่ออายุ ${label}`,
      html: curLine,
      defaultDays: 180
    });
    if (!days) return;
    try {
      const res = await renewImportLicense(licenseNo, invoiceNo, days);
      await loadAll();
      const newExp = res?.newExpiry ? formatThaiDate(new Date(res.newExpiry)) : '';
      toastSuccess(`ต่ออายุ ${label} อีก ${days} วันแล้ว${newExp ? ` — หมดอายุ ${newExp}` : ''}`);
    } catch (err) {
      const msg = err.message || 'ต่ออายุไม่สำเร็จ';
      setLoadError(msg);
      toastError(msg);
    }
  }
  const filtered = useMemo(() => {
    let rows = items;
    if (selectedLot) {
      const [licenseNo, invoiceNo] = selectedLot.split('|');
      rows = rows.filter(r => r.LicenseNo === licenseNo && r.InvoiceNo === invoiceNo);
    }
    if (countryFilter !== ALL_COUNTRIES) {
      rows = rows.filter(r => countryKey(r.ExportCountry) === countryFilter);
    }
    if (expiryFilter === COMPLETED_FILTER) {
      rows = rows.filter(isLicenseCompleted);
    } else if (expiryFilter !== 'all') {
      rows = rows.filter(r => !isLicenseCompleted(r) && computeLicenseExpiry(r.IssueDate).status === expiryFilter);
    }
    const term = search.trim().toLowerCase();
    if (term) {
      rows = rows.filter(r => (r.MachineNo || '').toLowerCase().includes(term) || (r.ProductionNo || '').toLowerCase().includes(term) || (r.LicenseNo || '').toLowerCase().includes(term) || (r.InvoiceNo || '').toLowerCase().includes(term) || (r.DeclarationNo || '').toLowerCase().includes(term) || (r.Model || '').toLowerCase().includes(term) || (r.ExportCountry || '').toLowerCase().includes(term));
    }
    rows = [...rows].sort((a, b) => (a.SortOrder || 0) - (b.SortOrder || 0) || a.ID - b.ID);
    return rows;
  }, [items, selectedLot, countryFilter, expiryFilter, search, today]);
  const countryOptions = useMemo(() => buildCountryOptions(items.map(r => r.ExportCountry)), [items]);
  const filterActive = !!selectedLot || countryFilter !== ALL_COUNTRIES || expiryFilter !== 'all' || search.trim() !== '';

  useEffect(() => {
    if (countryFilter !== ALL_COUNTRIES && !countryOptions.some(o => o.value === countryFilter)) {
      setCountryFilter(ALL_COUNTRIES);
    }
  }, [countryOptions, countryFilter]);
  const expiryOptions = useMemo(() => [{
    value: 'all',
    label: 'ทุกสถานะวันหมดอายุ'
  }, {
    value: EXPIRY_STATUS.NO_DATE,
    label: STATUS_LABEL[EXPIRY_STATUS.NO_DATE]
  }, {
    value: EXPIRY_STATUS.EXPIRING,
    label: STATUS_LABEL[EXPIRY_STATUS.EXPIRING]
  }, {
    value: EXPIRY_STATUS.EXPIRED,
    label: STATUS_LABEL[EXPIRY_STATUS.EXPIRED]
  }, {
    value: EXPIRY_STATUS.VALID,
    label: STATUS_LABEL[EXPIRY_STATUS.VALID]
  }, {
    value: COMPLETED_FILTER,
    label: COMPLETED_LABEL
  }], []);
  const lotOptions = useMemo(() => {
    const opts = [{
      value: '',
      label: 'ทุกใบอนุญาต'
    }];
    const lotsPerLicense = new Map();
    summary.forEach(s => lotsPerLicense.set(s.LicenseNo, (lotsPerLicense.get(s.LicenseNo) || 0) + 1));
    summary.forEach(s => {
      const completedAll = s.Total > 0 && s.CompletedCount >= s.Total;
      const extra = lotsPerLicense.get(s.LicenseNo) > 1 ? `Invoice ${s.InvoiceNo || '—'}` : '';
      opts.push({
        value: `${s.LicenseNo}|${s.InvoiceNo}`,
        label: licenseOptionLabel(s.LicenseNo, s.Total, s.CompletedCount, extra),
        suffix: completedAll ? <CompletedOptionIcon /> : null
      });
    });
    return opts;
  }, [summary]);
  const counts = useMemo(() => ({
    total: items.length,
    licenses: new Set(items.map(r => r.LicenseNo).filter(Boolean)).size,
    invoices: new Set(items.map(r => r.InvoiceNo).filter(Boolean)).size,
    completed: items.filter(isLicenseCompleted).length
  }), [items]);

  const expiryCounts = useMemo(() => {
    const worst = new Map();
    for (const row of items) {
      if (isLicenseCompleted(row)) continue;
      const key = (row.LicenseNo || '').trim();
      if (!key) continue;
      const exp = row.ExpireDate ? computeExpireStatus(row.ExpireDate, 30) : computeLicenseExpiry(row.IssueDate);
      if (exp.status !== EXPIRY_STATUS.EXPIRED && exp.status !== EXPIRY_STATUS.EXPIRING) continue;
      if (worst.get(key) === EXPIRY_STATUS.EXPIRED) continue;
      worst.set(key, exp.status);
    }
    let expiring = 0;
    let expired = 0;
    for (const status of worst.values()) {
      if (status === EXPIRY_STATUS.EXPIRED) expired++;else expiring++;
    }
    return {
      expiring,
      expired
    };
  }, [items, today]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages));
  }
  const currentLot = summary.find(s => `${s.LicenseNo}|${s.InvoiceNo}` === selectedLot);

  const {
    selected,
    toggleOne,
    replaceWith: replaceSelection,
    clear: clearSelection,
    toggleAll,
    invert: invertSelection,
    allSelected,
    someSelected
  } = useRowSelection(filtered);
  const selectedRows = useMemo(() => filtered.filter(r => selected.has(r.ID)), [filtered, selected]);
  const selectedLockedCount = useMemo(() => selectedRows.filter(r => r.Locked).length, [selectedRows]);
  const lotValues = useMemo(() => summary.map(sm => `${sm.LicenseNo}|${sm.InvoiceNo}`), [summary]);
  const {
    checked: checkedLots,
    toggle: toggleLotCheck,
    setAll: setAllLotChecks,
    clear: clearLotChecks
  } = useCheckedSet(lotValues);
  const checkedLotRows = useMemo(() => checkedLots.size === 0 ? [] : items.filter(r => checkedLots.has(`${r.LicenseNo}|${r.InvoiceNo}`)), [items, checkedLots]);
  const checkedLotDone = useMemo(() => checkedLotRows.filter(isLicenseCompleted).length, [checkedLotRows]);
  function lotLabelOf(value) {
    const [licenseNo, invoiceNo] = value.split('|');
    return licenseNo || (invoiceNo ? `Invoice ${invoiceNo}` : '(ไม่มีเลขใบอนุญาต)');
  }

  async function completeCheckedLots(completed) {
    const targets = completed ? checkedLotRows.filter(r => !isLicenseCompleted(r)) : checkedLotRows.filter(isLicenseCompleted);
    if (targets.length === 0) return;
    const n = checkedLots.size;
    const ok = await confirmComplete({
      title: completed ? `ทำเครื่องหมายเสร็จสิ้น ${fmtCount(n)} ใบอนุญาต?` : `ยกเลิกสถานะเสร็จสิ้น ${fmtCount(n)} ใบอนุญาต?`,
      html: `${fmtCount(targets.length)} เครื่อง`,
      danger: !completed
    });
    if (!ok) return;
    setCompleting(true);
    try {
      await setImportLicenseComplete({
        ids: targets.map(r => r.ID),
        completed
      });
      clearLotChecks();
      await loadAll();
      toastSuccess(completed ? `ปิดงาน ${fmtCount(n)} ใบอนุญาตแล้ว (${fmtCount(targets.length)} เครื่อง) — หยุดนับวันหมดอายุ` : `ยกเลิกสถานะเสร็จสิ้น ${fmtCount(n)} ใบอนุญาตแล้ว`);
    } catch (err) {
      toastError(err.message || 'อัปเดตสถานะไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }

  async function deleteCheckedLots() {
    const lots = [...checkedLots];
    if (lots.length === 0) return;
    const names = lots.slice(0, 5).map(lotLabelOf).join(', ') + (lots.length > 5 ? ` และอีก ${lots.length - 5} ใบ` : '');
    const ok = await confirmDelete({
      title: `ลบ ${fmtCount(lots.length)} ใบอนุญาต?`,
      text: `${names} (${fmtCount(checkedLotRows.length)} เครื่อง) — ลบแล้วกู้คืนไม่ได้`,
      confirmText: `ลบ ${fmtCount(lots.length)} ใบ`
    });
    if (!ok) return;
    setCompleting(true);
    let deleted = 0;
    let failed = 0;
    try {
      for (const lot of lots) {
        const [licenseNo, invoiceNo] = lot.split('|');
        try {
          const res = await clearImportLicense(licenseNo, invoiceNo);
          deleted += res?.deleted ?? 0;
        } catch {
          failed += 1;
        }
      }
      if (lots.includes(selectedLot)) setSelectedLot('');
      clearLotChecks();
      clearSelection();
      await loadAll();
      if (failed > 0) toastError(`ลบไม่สำเร็จ ${fmtCount(failed)} ใบ`);else toastSuccess(`ลบ ${fmtCount(lots.length)} ใบอนุญาตแล้ว (${fmtCount(deleted)} เครื่อง)`);
    } finally {
      setCompleting(false);
    }
  }

  async function applyBulkDelete() {
    const deletable = selectedRows.filter(r => !r.Locked);
    const lockedCount = selectedRows.length - deletable.length;
    if (deletable.length === 0) {
      toastError('รายการที่เลือกสแกนแล้วทั้งหมด ลบไม่ได้');
      return;
    }
    const ok = await confirmDelete({
      title: `ลบ ${fmtCount(deletable.length)} รายการที่เลือก?`,
      text: `ลบออกจากระบบแล้วกู้คืนไม่ได้${lockedCount > 0 ? ` (ข้าม ${fmtCount(lockedCount)} รายการที่สแกนแล้ว)` : ''}`,
      confirmText: `ลบ ${fmtCount(deletable.length)} รายการ`
    });
    if (!ok) return;
    setCompleting(true);
    try {
      const res = await bulkDeleteImportLicenseItems(deletable.map(r => r.ID));
      const skipped = res?.skipped?.length ?? 0;
      clearSelection();
      await loadAll();
      toastSuccess(`ลบแล้ว ${fmtCount(res?.deleted ?? deletable.length)} รายการ${skipped > 0 ? ` · ข้าม ${fmtCount(skipped)} รายการที่สแกนแล้ว` : ''}`);
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }

  async function applyComplete(completed) {
    const targets = completed ? selectedRows.filter(r => !isLicenseCompleted(r)) : selectedRows.filter(isLicenseCompleted);
    if (targets.length === 0) return;
    const ok = await confirmComplete({
      title: completed ? `ต้องการทำเครื่องหมายเสร็จสิ้น ${fmtCount(targets.length)} รายการ` : `ต้องการยกเลิกสถานะเสร็จสิ้น ${fmtCount(targets.length)} รายการ`,
      danger: !completed
    });
    if (!ok) return;
    setCompleting(true);
    try {
      await setImportLicenseComplete({
        ids: targets.map(r => r.ID),
        completed
      });
      clearSelection();
      await loadAll();
      toastSuccess(completed ? `ทำเครื่องหมายเสร็จสิ้น ${fmtCount(targets.length)} รายการแล้ว — หยุดนับวันหมดอายุ` : `ยกเลิกสถานะเสร็จสิ้น ${fmtCount(targets.length)} รายการแล้ว`);
    } catch (err) {
      toastError(err.message || 'อัปเดตสถานะไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }

  async function handleCompleteLot(lot, completed) {
    const licenseNo = lot?.LicenseNo ?? '';
    const invoiceNo = lot?.InvoiceNo ?? '';
    const label = licenseNo || (invoiceNo ? `Invoice ${invoiceNo}` : 'ล็อตนี้ (ไม่มีเลขใบอนุญาต)');
    const ok = await confirmComplete({
      title: completed ? `ต้องการทำเครื่องหมายเสร็จสิ้นทั้งใบ ${label}` : `ต้องการยกเลิกสถานะเสร็จสิ้นทั้งใบ ${label}`,
      danger: !completed
    });
    if (!ok) return;
    setCompleting(true);
    try {
      const res = await setImportLicenseComplete({
        licenseNo,
        invoiceNo,
        completed
      });
      clearSelection();
      await loadAll();
      toastSuccess(completed ? `ปิดงาน ${label} แล้ว (${res?.updated ?? 0} เครื่อง) — หยุดนับวันหมดอายุ` : `ยกเลิกสถานะเสร็จสิ้น ${label} แล้ว (${res?.updated ?? 0} เครื่อง)`);
    } catch (err) {
      toastError(err.message || 'อัปเดตสถานะไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }

  async function handleToggleRowComplete(row) {
    const completed = !isLicenseCompleted(row);
    setCompleting(true);
    try {
      await setImportLicenseComplete({
        ids: [row.ID],
        completed
      });
      setDetailRow(null);
      await loadAll();
      toastSuccess(completed ? `${row.MachineNo || 'รายการนี้'} เสร็จสิ้นแล้ว — หยุดนับวันหมดอายุ` : `ยกเลิกสถานะเสร็จสิ้นของ ${row.MachineNo || 'รายการนี้'} แล้ว`);
    } catch (err) {
      toastError(err.message || 'อัปเดตสถานะไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }
  const lotCompletedAll = currentLot ? currentLot.CompletedCount >= currentLot.Total && currentLot.Total > 0 : false;
  return <AppShell navItems={WH_NAV_ITEMS} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Import License</h2>
        </div>
      </div>

      {loadError && <p className="form-error" role="alert">
          {loadError}
        </p>}

      <div className="wh-upload-card">
        <div className="fdz-row">
          <FileDropZone file={file} onSelect={f => {
          setFile(f);
          setUploadMsg(null);
          setPreviewData(null);
        }} accept=".xlsx,.xls,.csv" label="อัปโหลดบัญชีใบอนุญาตนำเข้า" disabled={uploading} />
          <button className="wh-modal-cancel" onClick={handlePreview} disabled={previewing || uploading || !file}>
            {previewing ? 'กำลังตรวจสอบ...' : 'ตรวจสอบก่อนอัปโหลด'}
          </button>
          <button className="wh-issue-btn" onClick={handleUpload} disabled={uploading || !file}>
            {uploading ? 'กำลังอัปโหลด...' : 'อัปโหลด'}
          </button>
        </div>

        {previewData && (previewData.summary ? <ChangePreview result={previewData} /> : <PreviewResult result={previewData} />)}

        {uploadMsg?.success && <p className="upload-card-msg upload-card-msg-ok wh-upload-msg">{uploadMsg.success}</p>}
        {uploadMsg?.error && <p className="upload-card-msg upload-card-msg-err wh-upload-msg">{uploadMsg.error}</p>}
        {uploadMsg?.problems?.length > 0 && <ul className="il-problem-list">
            {uploadMsg.problems.map((p, i) => <li key={i}>{p}</li>)}
          </ul>}
      </div>

      <div className="dash-stats-row wh-stats-row il-stats-row-5">
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <StatLabel parts={['เครื่องในบัญชี', 'ทั้งหมด']} />
            <span className="dash-stat-icon dash-icon-blue">
              <Squares2X2Icon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.total}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <StatLabel parts={['ใบอนุญาต', 'นำเข้า']} />
            <span className="dash-stat-icon dash-icon-red">
              <DocumentTextIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.licenses}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <StatLabel parts={['อินวอยซ์', 'นำเข้า']} />
            <span className="dash-stat-icon dash-icon-yellow">
              <ReceiptPercentIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.invoices}</div>
        </div>
        <div className="dash-stat-card il-stat-expiry">
          <div className="dash-stat-label">
            <StatLabel parts={['อายุ', 'ใบอนุญาต']} />
            <span className="dash-stat-icon dash-icon-orange">
              <ClockIcon className="size-4" />
            </span>
          </div>
          <div className="il-stat-expiry-split">
            <div className="il-stat-expiry-part il-stat-expiring">
              <div className="dash-stat-value">{expiryCounts.expiring}</div>
              <div className="dash-stat-note">ใกล้หมดอายุ</div>
            </div>
            <span className="il-stat-expiry-divider" aria-hidden="true" />
            <div className="il-stat-expiry-part il-stat-expired">
              <div className="dash-stat-value">{expiryCounts.expired}</div>
              <div className="dash-stat-note">หมดอายุแล้ว</div>
            </div>
          </div>
        </div>
        <div className="dash-stat-card il-stat-complete">
          <div className="dash-stat-label">
            <StatLabel parts={['เสร็จสิ้นแล้ว']} />
            <span className="dash-stat-icon dash-icon-green">
              <CheckBadgeIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.completed}</div>
        </div>
      </div>

      {summary.length > 0 && <div className="il-lot-filter">
          <label className="il-lot-filter-label">ใบอนุญาต</label>
          <div className="il-lot-filter-select">
            <LicenseCheckSelect value={selectedLot} onChange={setSelectedLot} options={lotOptions} allValue="" checked={checkedLots} onToggleCheck={toggleLotCheck} onCheckAll={setAllLotChecks} />
          </div>
        </div>}

      <LicenseActionBar licenseCount={checkedLots.size} machineCount={checkedLotRows.length} openCount={checkedLotRows.length - checkedLotDone} doneCount={checkedLotDone} busy={completing} onComplete={() => completeCheckedLots(true)} onUncomplete={() => completeCheckedLots(false)} onDelete={deleteCheckedLots} onClear={clearLotChecks} />

      {currentLot && <div className="wh-so-active-bar il-license-bar">
          <div className="il-lot-info">
            <div className="il-lot-info-text">
              <span className="wh-so-active-label">ใบอนุญาตนำเข้า</span>
              <h3 className="wh-so-active-name">{currentLot.LicenseNo || '(ไม่มีเลขใบอนุญาต)'}</h3>
              <span className="wh-subtitle">
                Invoice {currentLot.InvoiceNo || '—'} · ใบขนสินค้า {currentLot.DeclarationNo || '—'} · {currentLot.Total} เครื่อง
              </span>
            </div>
            {lotCompletedAll && <span className="il-complete-stamp" title="ปิดงานทั้งใบแล้ว">
                <CheckBadgeSolidIcon className="il-complete-stamp-icon" aria-hidden="true" />
                เสร็จสิ้นแล้ว
              </span>}
          </div>
          <div className="il-lot-actions">
            <button className={lotCompletedAll ? 'il-uncomplete-btn' : 'il-complete-btn'} onClick={() => handleCompleteLot(currentLot, !lotCompletedAll)} disabled={completing} title={lotCompletedAll ? 'กลับมานับวันหมดอายุและแจ้งเตือนใบนี้ตามปกติ' : 'ปิดงานทั้งใบ แล้วหยุดนับวันหมดอายุ'}>
              {lotCompletedAll ? <>
                  <ArrowPathIcon className="size-4" />
                  ยกเลิกเสร็จสิ้นทั้งใบ
                </> : <>
                  <CheckBadgeIcon className="size-4" />
                  ยืนยันเสร็จสิ้น
                </>}
            </button>
            <button className="wh-issue-btn il-renew-btn" onClick={() => handleRenewLicense(currentLot)}>
              ต่ออายุ
            </button>
            <button className="wh-modal-cancel" onClick={() => handleClearLicense(currentLot)}>
              ลบทั้งใบ
            </button>
          </div>
        </div>}

      <div className="tsf-history-toolbar">
        <div className="tsf-history-pagesize">
          <div className="wh-pagesize-select">
            <SelectField value={pageSize} onChange={setPageSize} options={[{
            value: 10,
            label: '10'
          }, {
            value: 25,
            label: '25'
          }, {
            value: 50,
            label: '50'
          }, {
            value: 100,
            label: '100'
          }]} />
          </div>
          entries per page
        </div>
        <div className="il-filter-search-group">
          <div className="wh-pagesize-select il-model-filter">
            <SelectField value={countryFilter} onChange={setCountryFilter} options={countryOptions} />
          </div>
          <div className="wh-pagesize-select il-model-filter">
            <SelectField value={expiryFilter} onChange={setExpiryFilter} options={expiryOptions} />
          </div>
          <input className="wh-search" type="text" placeholder="ค้นหา หมายเลขเครื่อง / หมายเลขการผลิต / ใบอนุญาต / อินวอยซ์ / ใบขนสินค้า" value={search} onChange={e => setSearch(e.target.value)} />
          {items.length > 0 && <button className="wh-btn-danger" onClick={handleClearAllImport}>
              ลบทุกใบอนุญาต
            </button>}
        </div>
      </div>

      <SelectionBar selectedRows={selectedRows} busy={completing} lockedCount={selectedLockedCount} onComplete={() => applyComplete(true)} onUncomplete={() => applyComplete(false)} onDelete={applyBulkDelete} onInvert={invertSelection} onClear={clearSelection} allSelected={allSelected} filterActive={filterActive} />

      <div className="wh-table-card">
        <table className="wh-table il-table-selectable">
          <thead>
            <tr>
              <th className="il-check-th">
                <SelectCheckbox checked={allSelected} indeterminate={someSelected} onChange={toggleAll} label={`เลือกทุกรายการ (${fmtCount(filtered.length)} รายการ)`} title={allSelected ? 'ยกเลิกการเลือกทั้งหมด' : `เลือกทุกรายการ ${fmtCount(filtered.length)} รายการ`} disabled={filtered.length === 0} />
              </th>
              <th>ลำดับ</th>
              <th>ตราอักษร</th>
              <th>แบบ/รุ่น</th>
              <th>เลขใบอนุญาตนำเข้า</th>
              <th>วันที่ออกใบอนุญาต</th>
              <th>หมดอายุ (6 เดือน)</th>
              <th>เลขอินวอยซ์นำเข้า</th>
              <th>เลขใบขนสินค้าขาเข้า</th>
              <th>จำนวน (เครื่อง)</th>
              <th>หมายเลขเครื่อง</th>
              <th>หมายเลขการผลิต</th>
              <th>หมายเหตุ</th>
              <th>ส่งออกไปประเทศ</th>
              <th>คอลัมน์เพิ่ม</th>
              <th>สถานะการสแกน</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={17} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>}
            {!loading && paged.map((row, i) => {
            const lock = {
              locked: !!row.Locked,
              lockReason: row.LockReason
            };
            const save = key => v => saveImportField(row, key, v);
            return <tr key={row.ID} className={(isLicenseCompleted(row) ? 'il-row-complete' : '') + (selected.has(row.ID) ? ' il-row-selected' : '') + (row.Locked ? ' ie-row-locked' : '')}>
                  <td className="il-check-td" data-label="เลือก">
                    <SelectCheckbox checked={selected.has(row.ID)} onChange={(_, shift) => toggleOne(row.ID, shift)} label={`เลือก ${row.MachineNo || 'รายการนี้'}`} title="Shift+คลิก เพื่อเลือกเป็นช่วง" />
                  </td>
                  <td className="wh-cell-head" data-label="ลำดับ">
                    {(page - 1) * pageSize + i + 1}
                  </td>
                  <td data-label="ตราอักษร">
                    <EditableCell readOnly {...lock} label="ตราอักษร" value={row.Brand} onSave={save('Brand')} />
                  </td>
                  <td data-label="แบบ/รุ่น">
                    <EditableCell readOnly {...lock} label="แบบ/รุ่น" value={row.Model} onSave={save('Model')} />
                  </td>
                  <td data-label="เลขใบอนุญาตนำเข้า">
                    <span className="il-license-cell">
                      <CompleteFlag show={isLicenseCompleted(row)} />
                      <EditableCell readOnly {...lock} label="เลขใบอนุญาตนำเข้า" value={row.LicenseNo} onSave={save('LicenseNo')} />
                    </span>
                  </td>
                  <td data-label="วันที่ออกใบอนุญาต">
                    <EditableCell readOnly {...lock} type="date" label="วันที่ออกใบอนุญาต" value={row.IssueDate} display={formatThaiDate(row.IssueDate)} onSave={save('IssueDate')} />
                  </td>
                  <td data-label="หมดอายุ (6 เดือน)">
                    <ExpiryCell row={row} issueDate={row.IssueDate} expireDate={row.ExpireDate} />
                  </td>
                  <td data-label="เลขอินวอยซ์นำเข้า">
                    <EditableCell readOnly {...lock} label="เลขอินวอยซ์นำเข้า" value={row.InvoiceNo} onSave={save('InvoiceNo')} />
                  </td>
                  <td data-label="เลขใบขนสินค้าขาเข้า">
                    <EditableCell readOnly {...lock} label="เลขใบขนสินค้าขาเข้า" value={row.DeclarationNo} onSave={save('DeclarationNo')} />
                  </td>
                  <td data-label="จำนวน (เครื่อง)">
                    <EditableCell readOnly {...lock} type="number" label="จำนวน (เครื่อง)" value={row.Qty} onSave={save('Qty')} />
                  </td>
                  <td className="il-mono" data-label="หมายเลขเครื่อง">
                    <EditableCell readOnly {...lock} mono label="หมายเลขเครื่อง" value={row.MachineNo} display={<strong>{row.MachineNo}</strong>} onSave={save('MachineNo')} />
                  </td>
                  <td className="il-mono" data-label="หมายเลขการผลิต">
                    <EditableCell readOnly {...lock} mono label="หมายเลขการผลิต" value={row.ProductionNo} onSave={save('ProductionNo')} />
                  </td>
                  <td data-label="หมายเหตุ">
                    <EditableCell readOnly {...lock} label="หมายเหตุ" value={row.Remark} onSave={save('Remark')} />
                  </td>
                  <td data-label="ส่งออกไปประเทศ">
                    <EditableCell readOnly {...lock} label="ส่งออกไปประเทศ" value={row.ExportCountry} display={row.ExportCountry || <span className="il-no-country">{NO_COUNTRY_LABEL}</span>} onSave={save('ExportCountry')} />
                  </td>
                  <td data-label="คอลัมน์เพิ่ม">
                    <ExtraColumnsCell json={row.extra_json} />
                  </td>
                  <td data-label="สถานะการสแกน">
                    {row.Locked ? <LockBadge locked variant="done" reason={row.LockReason} /> : <span className="ie-empty">ยังไม่ได้สแกน</span>}
                  </td>
                  <td className="wh-cell-action">
                    <div className="il-row-actions">
                      <button className="wh-modal-cancel" onClick={() => setDetailRow(row)}>
                        รายละเอียด
                      </button>
                      <button className="wh-btn-danger" disabled={row.Locked} title={row.Locked ? `ลบไม่ได้ — ${row.LockReason || 'สแกนผ่านแล้ว'}` : ''} onClick={() => handleDeleteRow(row)}>
                        ลบ
                      </button>
                    </div>
                  </td>
                </tr>;
          })}
            {!loading && paged.length === 0 && <tr>
                <td colSpan={17} className="wh-empty-cell">
                  ยังไม่มีข้อมูล
                </td>
              </tr>}
          </tbody>
        </table>
      </div>

      {!loading && filtered.length > 0 && <div className="tsf-pagination">
          <span className="wh-subtitle" style={{
        fontSize: 13
      }}>
            Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, filtered.length)} of{' '}
            {filtered.length} entries
          </span>
          <div className="tsf-pagination-buttons">
            <button className="wh-modal-cancel" onClick={() => goToPage(1)} disabled={page === 1}>
              <ChevronDoubleLeftIcon className="size-4" />
            </button>
            <button className="wh-modal-cancel" onClick={() => goToPage(page - 1)} disabled={page === 1}>
              <ChevronLeftIcon className="size-4" />
            </button>
            <span className="tsf-pagination-current">
              {page} / {totalPages}
            </span>
            <button className="wh-modal-cancel" onClick={() => goToPage(page + 1)} disabled={page === totalPages}>
              <ChevronRightIcon className="size-4" />
            </button>
            <button className="wh-modal-cancel" onClick={() => goToPage(totalPages)} disabled={page === totalPages}>
              <ChevronDoubleRightIcon className="size-4" />
            </button>
          </div>
        </div>}

      {detailRow && <ImportDetailModal row={detailRow} busy={completing} onToggleComplete={handleToggleRowComplete} onClose={() => setDetailRow(null)} />}
    </AppShell>;
}
function useDetailSheet(onClose) {
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;
  const scrollRef = useRef(null);
  useEffect(() => {
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const onKey = e => {
      if (e.key === 'Escape') onCloseRef.current?.();
    };
    window.addEventListener('keydown', onKey);

    const scroller = scrollRef.current;
    const overlay = scroller?.closest('.il-detail-overlay');
    const fromOutside = e => scroller && !scroller.contains(e.target);
    const onWheel = e => {
      if (!fromOutside(e)) return;
      e.preventDefault();
      scroller.scrollTop += e.deltaMode === 1 ? e.deltaY * 16 : e.deltaY;
    };
    let lastY = null;
    const onTouchStart = e => {
      lastY = fromOutside(e) && e.touches.length === 1 ? e.touches[0].clientY : null;
    };
    const onTouchMove = e => {
      if (lastY === null) return;
      const y = e.touches[0].clientY;
      scroller.scrollTop += lastY - y;
      lastY = y;
      e.preventDefault();
    };
    const onTouchEnd = () => {
      lastY = null;
    };
    if (overlay) {
      overlay.addEventListener('wheel', onWheel, {
        passive: false
      });
      overlay.addEventListener('touchstart', onTouchStart, {
        passive: true
      });
      overlay.addEventListener('touchmove', onTouchMove, {
        passive: false
      });
      overlay.addEventListener('touchend', onTouchEnd);
      overlay.addEventListener('touchcancel', onTouchEnd);
    }
    return () => {
      document.body.style.overflow = prevOverflow;
      window.removeEventListener('keydown', onKey);
      if (overlay) {
        overlay.removeEventListener('wheel', onWheel);
        overlay.removeEventListener('touchstart', onTouchStart);
        overlay.removeEventListener('touchmove', onTouchMove);
        overlay.removeEventListener('touchend', onTouchEnd);
        overlay.removeEventListener('touchcancel', onTouchEnd);
      }
    };
  }, []);
  return scrollRef;
}
function ImportDetailModal({
  row,
  onClose,
  onToggleComplete,
  busy = false
}) {
  const completed = isLicenseCompleted(row);
  const scrollRef = useDetailSheet(onClose);
  const item = (label, value) => <div className="wh-detail-item">
      <span className="wh-detail-label">{label}</span>
      <span className="wh-detail-value">{value === 0 || value ? value : '—'}</span>
    </div>;
  const exp = computeLicenseExpiry(row.IssueDate);
  const CONFIRM_LABEL = {
    CONFIRMED: 'ยืนยันแล้ว',
    PENDING: 'รอยืนยัน',
    REJECTED: 'ไม่ผ่าน'
  };
  const confirmLabel = CONFIRM_LABEL[row.ConfirmStatus] || row.ConfirmStatus;
  let extraEntries = [];
  try {
    const obj = row.extra_json ? JSON.parse(row.extra_json) : null;
    if (obj) extraEntries = Object.entries(obj);
  } catch {
    extraEntries = [];
  }
  return <div className="wh-modal-overlay il-detail-overlay" onClick={onClose}>
      <div className="wh-modal wh-detail-modal il-detail-sheet" role="dialog" aria-modal="true" aria-labelledby="il-import-detail-title" onClick={e => e.stopPropagation()}>
        <button type="button" className="wh-detail-close" onClick={onClose} aria-label="ปิด">
          <XMarkIcon className="size-4" />
        </button>

        <div className="wh-detail-header">
          <span className="wh-detail-header-icon">
            <DocumentTextIcon className="size-5" />
          </span>
          <div>
            <h3 className="wh-modal-title" id="il-import-detail-title">รายละเอียดใบอนุญาตนำเข้า</h3>
            <span className="wh-detail-header-sub">{row.Model || row.Brand || '—'}</span>
          </div>
        </div>

        <div className="il-detail-scroll" ref={scrollRef}>

        <div className="wh-detail-section">
          <span className="wh-detail-section-title">
            <CubeIcon className="size-4" /> ข้อมูลเครื่อง
          </span>
          <div className="wh-detail-grid">
            {item('หมายเลขเครื่อง', row.MachineNo)}
            {item('หมายเลขการผลิต', row.ProductionNo)}
            {item('ตราอักษร', row.Brand)}
            {item('แบบ/รุ่น', row.Model)}
            {item('จำนวน (เครื่อง)', row.Qty)}
            {item('ส่งออกไปประเทศ', row.ExportCountry || NO_COUNTRY_LABEL)}
          </div>
        </div>

        <div className="wh-detail-divider" />

        <div className="wh-detail-section">
          <span className="wh-detail-section-title">
            <DocumentTextIcon className="size-4" /> ข้อมูลใบอนุญาต
          </span>
          <div className="wh-detail-grid">
            {item('เลขใบอนุญาตนำเข้า', row.LicenseNo)}
            {item('เลขอินวอยซ์นำเข้า', row.InvoiceNo)}
            {item('เลขใบขนสินค้าขาเข้า', row.DeclarationNo)}
            {item('วันที่ออกใบอนุญาต', row.IssueDate ? formatThaiDate(row.IssueDate) : '')}
            {item('วันหมดอายุ (6 เดือน)', exp.hasDate ? formatThaiDate(exp.expiryDate) : '')}
            <div className="wh-detail-item">
              <span className="wh-detail-label">สถานะอายุ</span>
              <span className="wh-detail-value il-detail-status">
                {completed ? <>
                    <span className="il-badge il-badge-complete">
                      <CheckBadgeIcon className="size-3.5" />
                      {COMPLETED_LABEL}
                    </span>
                    {exp.hasDate && <span className="il-detail-days">หมดอายุ {formatThaiDate(exp.expiryDate)}</span>}
                  </> : <>
                    <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
                    {exp.hasDate && <span className="il-detail-days">{daysLeftLabel(exp.daysLeft)}</span>}
                  </>}
              </span>
            </div>
            {item('หมายเหตุ', row.Remark)}
          </div>
        </div>

        <div className="wh-detail-divider" />

        <div className="wh-detail-section">
          <span className="wh-detail-section-title">
            <ShieldCheckIcon className="size-4" /> สถานะการยืนยัน
          </span>
          <div className="wh-detail-grid">
            {item('สถานะ', confirmLabel)}
            {item('ผู้ยืนยัน', row.ConfirmedBy)}
            {item('วันเวลาที่ยืนยัน', row.ConfirmedDatetime ? formatThaiDate(row.ConfirmedDatetime) : '')}
            {item('สถานะปิดงาน', completed ? COMPLETED_LABEL : 'ยังไม่เสร็จสิ้น')}
            {completed && item('ผู้กดเสร็จสิ้น', row.CompletedBy)}
            {completed && item('วันที่กดเสร็จสิ้น', row.CompletedAt ? formatThaiDate(row.CompletedAt) : '')}
          </div>
        </div>

        {extraEntries.length > 0 && <>
            <div className="wh-detail-divider" />
            <div className="wh-detail-section">
              <span className="wh-detail-section-title">
                <RectangleStackIcon className="size-4" /> คอลัมน์เพิ่มจากไฟล์
              </span>
              <div className="wh-detail-grid">
                {extraEntries.map(([k, v]) => item(String(k).replace(/^\[\+\]\s*/, ''), v))}
              </div>
            </div>
          </>}

        <div className="wh-detail-meta">
          <span>
            <TagIcon className="size-3.5" /> ไฟล์ {row.FileName || '—'}
          </span>
          <span>
            <ClockIcon className="size-3.5" /> อัปโหลดเมื่อ {row.UploadDate ? formatThaiDate(row.UploadDate) : '—'}
          </span>
        </div>

        </div>

        <div className="wh-modal-actions il-modal-actions">
          {onToggleComplete && <button type="button" className={completed ? 'il-uncomplete-btn' : 'il-complete-btn'} onClick={() => onToggleComplete(row)} disabled={busy}>
              {completed ? <>
                  <ArrowPathIcon className="size-4" />
                  ยกเลิกสถานะเสร็จสิ้น
                </> : <>
                  <CheckBadgeIcon className="size-4" />
                  ทำเครื่องหมายเสร็จสิ้น
                </>}
            </button>}
          <button className="wh-modal-cancel" onClick={onClose}>
            ปิด
          </button>
        </div>
      </div>
    </div>;
}
function computeExpireStatus(expireRaw, withinDays = 30) {
  if (!expireRaw) {
    return {
      hasDate: false,
      expiryDate: null,
      daysLeft: null,
      status: EXPIRY_STATUS.NO_DATE
    };
  }
  const exp = new Date(expireRaw);
  if (Number.isNaN(exp.getTime())) {
    return {
      hasDate: false,
      expiryDate: null,
      daysLeft: null,
      status: EXPIRY_STATUS.NO_DATE
    };
  }
  const atMidnight = d => new Date(d.getFullYear(), d.getMonth(), d.getDate());
  const today = atMidnight(new Date());
  const expDay = atMidnight(exp);
  const daysLeft = Math.round((expDay - today) / 86400000);
  let status;
  if (daysLeft < 0) status = EXPIRY_STATUS.EXPIRED;else if (daysLeft <= withinDays) status = EXPIRY_STATUS.EXPIRING;else status = EXPIRY_STATUS.VALID;
  return {
    hasDate: true,
    expiryDate: expDay,
    daysLeft,
    status
  };
}
function ExportExpiryCell({
  expireDate
}) {
  const exp = computeExpireStatus(expireDate);
  return <div className="il-expiry-cell">
      <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
      {exp.hasDate && <>
          <span>{formatThaiDate(exp.expiryDate)}</span>
          <span className="il-expiry-days">{daysLeftLabel(exp.daysLeft)}</span>
        </>}
    </div>;
}
const EXTRA_COUNTRY_KEYS = ['country', 'countryname', 'exportcountry', 'ประเทศ', 'ปลายทาง', 'ส่งออกไปประเทศ'];
const EXTRA_COL_LIMIT = 25;
function normExtraKey(k) {
  return String(k).replace(/^\[\+\]\s*/, '').toLowerCase().replace(/[\s_./-]/g, '');
}
function extraLabel(k) {
  return String(k).replace(/^\[\+\]\s*/, '').trim();
}
function parseExtraJson(json) {
  if (!json) return {};
  try {
    const obj = JSON.parse(json);
    if (!obj || typeof obj !== 'object') return {};
    const out = {};
    Object.entries(obj).forEach(([k, v]) => {
      if (EXTRA_COUNTRY_KEYS.includes(normExtraKey(k))) return;
      const val = String(v ?? '').trim();
      if (val) out[extraLabel(k)] = val;
    });
    return out;
  } catch {
    return {};
  }
}
function collectExtraColumns(rows) {
  const counts = new Map();
  rows.forEach(r => {
    Object.keys(parseExtraJson(r.extra_json)).forEach(label => {
      counts.set(label, (counts.get(label) || 0) + 1);
    });
  });
  const labels = Array.from(counts.entries()).sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).map(e => e[0]);
  return {
    spread: labels.slice(0, EXTRA_COL_LIMIT),
    overflow: labels.slice(EXTRA_COL_LIMIT)
  };
}
function computeExportExpiry(row, withinDays = 7) {
  return computeExportLicenseDates(row, {
    withinDays
  });
}
function ExportOneMonthExpiryCell({
  row
}) {
  const exp = computeExportExpiry(row);
  if (isLicenseCompleted(row)) return <CompleteBadge row={row} date={exp.expiryDate} />;
  return <div className="il-expiry-cell">
      <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
      {exp.hasDate && <>
          <span>{formatThaiDate(exp.expiryDate)}</span>
          <span className="il-expiry-days">{daysLeftLabel(exp.daysLeft)}</span>
        </>}
    </div>;
}
function ExportLeadTimeCell({
  row
}) {
  const exp = computeExportExpiry(row);
  if (isLicenseCompleted(row)) return <CompleteBadge row={row} date={exp.leadDate} dateLabel="ครบกำหนดยื่น" />;
  if (!exp.hasDate) {
    return <div className="il-expiry-cell">
        <span className={LEAD_BADGE_CLASS[LEAD_STATUS.NO_DATE]}>
          {LEAD_STATUS_LABEL[LEAD_STATUS.NO_DATE]}
        </span>
      </div>;
  }
  return <div className="il-expiry-cell">
      <span className={leadBadgeClass(exp)}>{LEAD_STATUS_LABEL[exp.leadStatus]}</span>
      <span>{formatThaiDate(exp.leadDate)}</span>
      <span className={'il-expiry-days' + (exp.leadAlert ? ' il-lead-days-alert' : '')}>
        {leadDaysLabel(exp.leadDaysLeft)}
      </span>
    </div>;
}
// สถานะการประกอบ (Export License) — ดูจากข้อมูล MFG Assembly ในระบบ (Link.MFGStatus)
// แสดงเฉพาะในตาราง ไม่เกี่ยวกับไฟล์ที่อัปโหลด
const ASSEMBLY_STATUS_META = {
  MATCHED: {
    label: 'ประกอบแล้ว',
    cls: 'il-asm-badge il-asm-ok',
    icon: 'ok'
  },
  NOT_MATCHED: {
    label: 'รอยืนยันการประกอบ',
    cls: 'il-asm-badge il-asm-warn'
  },
  DUPLICATE: {
    label: 'ข้อมูลซ้ำ',
    cls: 'il-asm-badge il-asm-bad'
  },
  RETIRED_FORMAT: {
    label: 'รูปแบบเดิมถูกยกเลิก',
    cls: 'il-asm-badge il-asm-bad'
  }
};
function AssemblyStatusCell({
  row
}) {
  const link = row?.Link || {};
  if (!link.MFGMatched) return <span className="ie-empty">ยังไม่ประกอบ</span>;
  const meta = ASSEMBLY_STATUS_META[link.MFGStatus] || {
    label: link.MFGStatus || 'ไม่ทราบสถานะ',
    cls: 'il-asm-badge il-asm-warn'
  };
  const title = link.MFGMachineNo ? `ประกอบกับเครื่อง ${link.MFGMachineNo}` : undefined;
  return <span className={meta.cls} title={title}>
      {meta.icon === 'ok' && <CheckCircleIcon className="size-3.5" aria-hidden="true" />}
      {meta.label}
    </span>;
}

function ExportTraceModal({
  row,
  country,
  onClose,
  onToggleComplete,
  busy = false
}) {
  const completed = isLicenseCompleted(row);
  const scrollRef = useDetailSheet(onClose);
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState(null);
  const [err, setErr] = useState(null);
  useEffect(() => {
    let alive = true;
    setLoading(true);
    setErr(null);
    getExportLicenseTrace(row.ID).then(d => alive && setData(d)).catch(e => alive && setErr(e.message || 'โหลดข้อมูลเชื่อมโยงไม่สำเร็จ')).finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [row.ID]);
  const expiryInfo = computeExportExpiry(row);
  const item = (label, value) => value ? <div className="wh-detail-item">
        <span className="wh-detail-label">{label}</span>
        <span className="wh-detail-value">{value}</span>
      </div> : null;
  const itemAlways = (label, value) => <div className="wh-detail-item">
      <span className="wh-detail-label">{label}</span>
      <span className="wh-detail-value">{value || '—'}</span>
    </div>;
  return <div className="wh-modal-overlay il-detail-overlay" onClick={onClose}>
      <div className="wh-modal wh-detail-modal il-detail-sheet" role="dialog" aria-modal="true" aria-labelledby="il-export-detail-title" onClick={e => e.stopPropagation()}>
        <button type="button" className="wh-detail-close" onClick={onClose} aria-label="ปิด">
          <XMarkIcon className="size-4" />
        </button>

        <div className="wh-detail-header">
          <span className="wh-detail-header-icon">
            <TruckIcon className="size-5" />
          </span>
          <div>
            <h3 className="wh-modal-title" id="il-export-detail-title">รายละเอียดใบอนุญาตส่งออก</h3>
            <span className="wh-detail-header-sub">{row.MachineNo || row.ITControllerNo || '—'}</span>
          </div>
        </div>

        <div className="il-detail-scroll" ref={scrollRef}>

        <div className="wh-detail-section">
          <span className="wh-detail-section-title">
            <CubeIcon className="size-4" /> ข้อมูลชิ้นงาน
          </span>
          <div className="wh-detail-grid">
            {item('Machine No', row.MachineNo)}
            {item('IT Controller S/N', row.ITControllerNo)}
            {itemAlways('Serial Number', data?.masterData?.SerialNo)}
            {item('ประเทศปลายทาง', country || NO_COUNTRY_LABEL)}
          </div>
        </div>

        <div className="wh-detail-divider" />

        <div className="wh-detail-section">
          <span className="wh-detail-section-title">
            <DocumentTextIcon className="size-4" /> ข้อมูลใบขนส่งออก
          </span>
          <div className="wh-detail-grid">
            {item('Invoice No.', row.InvoiceNo)}
            {item('Invoice Date', row.InvoiceDate ? formatThaiDate(row.InvoiceDate) : '')}
            {item('Export Entry', row.ExportEntry)}
            {item('Export License', row.ExportLicenseNo)}
            {itemAlways('Import License', row.ImportLicenseNo)}
            {item('วันที่นำออกใบอนุญาต', row.IssueDate ? formatThaiDate(row.IssueDate) : '')}
            <div className="wh-detail-item">
              <span className="wh-detail-label">หมดอายุ (1 เดือน)</span>
              <span className="wh-detail-value il-detail-status">
                {completed ? <>
                    <span className="il-badge il-badge-complete">
                      <CheckBadgeIcon className="size-3.5" />
                      {COMPLETED_LABEL}
                    </span>
                    {expiryInfo.hasDate && <span className="il-detail-days">หมดอายุ {formatThaiDate(expiryInfo.expiryDate)}</span>}
                  </> : <>
                    <span className={EXPIRY_BADGE_CLASS[expiryInfo.status]}>{STATUS_LABEL[expiryInfo.status]}</span>
                    {expiryInfo.hasDate && <span className="il-detail-days">{formatThaiDate(expiryInfo.expiryDate)} · {daysLeftLabel(expiryInfo.daysLeft)}</span>}
                  </>}
              </span>
            </div>
            <div className="wh-detail-item">
              <span className="wh-detail-label">{`Lead time (${EXPORT_LICENSE_LEAD_DAYS} วัน)`}</span>
              <span className="wh-detail-value il-detail-status">
                {completed ? <>
                    <span className="il-badge il-badge-complete">
                      <CheckBadgeIcon className="size-3.5" />
                      {COMPLETED_LABEL}
                    </span>
                    {expiryInfo.hasDate && <span className="il-detail-days">ครบกำหนดยื่น {formatThaiDate(expiryInfo.leadDate)}</span>}
                  </> : <>
                    <span className={leadBadgeClass(expiryInfo)}>{LEAD_STATUS_LABEL[expiryInfo.leadStatus]}</span>
                    {expiryInfo.hasDate && <span className="il-detail-days">{formatThaiDate(expiryInfo.leadDate)} · {leadDaysLabel(expiryInfo.leadDaysLeft)}</span>}
                  </>}
              </span>
            </div>
            {completed && item('ผู้กดเสร็จสิ้น', row.CompletedBy)}
            {completed && item('วันที่กดเสร็จสิ้น', row.CompletedAt ? formatThaiDate(row.CompletedAt) : '')}
            {item("Date Ass'y", row.AssemblyDate ? formatThaiDate(row.AssemblyDate) : '')}
            {item('Remark', row.Remark)}
          </div>
        </div>

        {loading && <p className="il-detail-note">กำลังโหลดข้อมูลที่เชื่อมโยง...</p>}
        {err && <p className="il-detail-note il-detail-note-err">{err}</p>}

        {!loading && !err && <>
            <div className="wh-detail-divider" />
            <div className="wh-detail-section">
              <span className="wh-detail-section-title">
                <ShieldCheckIcon className="size-4" /> Import License
              </span>
              {data?.importLicense ? <div className="wh-detail-grid">
                  {item('เลขใบอนุญาตนำเข้า', data.importLicense.LicenseNo)}
                  {item('Invoice นำเข้า', data.importLicense.InvoiceNo)}
                  {item('รุ่น', data.importLicense.Model)}
                  {item('ประเทศส่งออก', data.importLicense.ExportCountry)}
                  {item('สถานะยืนยัน', data.importLicense.ConfirmStatus)}
                  {itemAlways('ผู้ยืนยัน (WH)', data.importLicense.ConfirmedBy)}
                  {itemAlways('วันที่เช็ค', data.importLicense.ConfirmedDatetime ? formatThaiDate(data.importLicense.ConfirmedDatetime) : '')}
                </div> : <p className="il-detail-note">
                  ไม่พบใบอนุญาตนำเข้า
                </p>}
            </div>

            {data?.mfgAssembly && <>
                <div className="wh-detail-divider" />
                <div className="wh-detail-section">
                  <span className="wh-detail-section-title">
                    <WrenchScrewdriverIcon className="size-4" /> MFG Assembly (ผลตรวจตอนประกอบ)
                  </span>
                  <div className="wh-detail-grid">
                    {item('สถานะ', data.mfgAssembly.Status)}
                    {item('Machine No (ที่ประกอบ)', data.mfgAssembly.MachineNo)}
                    {itemAlways('วันที่ประกอบ', data.mfgAssembly.DateAssembly ? formatThaiDate(data.mfgAssembly.DateAssembly) : '')}
                    {itemAlways('Check By (MFG)', data.mfgAssembly.CreatedBy)}
                  </div>
                </div>
              </>}
          </>}

        </div>

        <div className="wh-modal-actions il-modal-actions">
          {onToggleComplete && <button type="button" className={completed ? 'il-uncomplete-btn' : 'il-complete-btn'} onClick={() => onToggleComplete(row)} disabled={busy}>
              {completed ? <>
                  <ArrowPathIcon className="size-4" />
                  ยกเลิกสถานะเสร็จสิ้น
                </> : <>
                  <CheckBadgeIcon className="size-4" />
                  ทำเครื่องหมายเสร็จสิ้น
                </>}
            </button>}
          <button className="wh-modal-cancel" onClick={onClose}>
            ปิด
          </button>
        </div>
      </div>
    </div>;
}
export function WHExportLicensePanel() {
  useDailyTick();
  const params = useAppParams();
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [licenseFilter, setLicenseFilter] = useState('all');
  const [countryFilter, setCountryFilter] = useState(ALL_COUNTRIES);
  const [expiryFilter, setExpiryFilter] = useState('all');
  const [traceRow, setTraceRow] = useState(null);
  const [file, setFile] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [msg, setMsg] = useState(null);
  const [previewData, setPreviewData] = useState(null);
  const [previewing, setPreviewing] = useState(false);
  const [pageSize, setPageSize] = useState(25);
  const [page, setPage] = useState(1);
  const [exportingXlsx, setExportingXlsx] = useState(false);
  const [completing, setCompleting] = useState(false);
  const [periodMode, setPeriodMode] = useState('all');
  const [periodAnchor, setPeriodAnchor] = useState('');
  const [countryByITC, setCountryByITC] = useState({});
  useEffect(() => {
    let cancelled = false;
    async function loadCountryMap() {
      try {
        const imports = await getImportLicenseItems();
        const map = {};
        (Array.isArray(imports) ? imports : []).forEach(it => {
          const key = String(it.MachineNo || '').trim();
          const country = String(it.ExportCountry || '').trim();
          if (key && country) map[key] = country;
        });
        if (!cancelled) setCountryByITC(map);
      } catch {}
    }
    loadCountryMap();
    return () => {
      cancelled = true;
    };
  }, []);
  function countryOf(r) {
    const direct = String(r.Country || '').trim();
    if (direct) return direct;
    try {
      const extra = r.extra_json ? JSON.parse(r.extra_json) : null;
      if (extra) {
        for (const [k, v] of Object.entries(extra)) {
          const nk = String(k).replace(/^\[\+\]\s*/, '').toLowerCase().replace(/[\s_./-]/g, '');
          if (['country', 'countryname', 'exportcountry', 'ประเทศ', 'ปลายทาง', 'ส่งออกไปประเทศ'].includes(nk)) {
            const val = String(v || '').trim();
            if (val) return val;
          }
        }
      }
    } catch {}
    const a = String(r.ITControllerNo || '').trim();
    return countryByITC[a] || '';
  }
  async function handleExportByCountry() {
    if (exportingXlsx) return;
    setExportingXlsx(true);
    try {
      const list = filtered;
      if (!list.length) {
        toastError('ไม่มีรายการให้ Export');
        return;
      }
      const groups = new Map();
      list.forEach(r => {
        const key = countryKey(countryOf(r));
        if (!groups.has(key)) groups.set(key, []);
        groups.get(key).push(r);
      });
      const countryNames = Array.from(groups.keys()).sort((a, b) => {
        if (a === NO_COUNTRY) return 1;
        if (b === NO_COUNTRY) return -1;
        return a.localeCompare(b);
      });
      const baseColumns = [{
        key: 'item',
        header: 'Item',
        type: 'number',
        width: 6
      }, {
        key: 'assemblyDate',
        header: "Date Ass'y",
        type: 'center',
        width: 14
      }, {
        key: 'issueDate',
        header: 'วันที่นำออกใบอนุญาต',
        type: 'center',
        width: 18
      }, {
        key: 'expiryDate',
        header: 'หมดอายุ (1 เดือน)',
        type: 'center',
        width: 18
      }, {
        key: 'leadTimeDate',
        header: `Lead time (${EXPORT_LICENSE_LEAD_DAYS} วัน)`,
        type: 'center',
        width: 24
      }, {
        key: 'completedDate',
        header: 'วันที่เสร็จสิ้น',
        type: 'center',
        width: 18
      }, {
        key: 'machineNo',
        header: 'Machine No',
        type: 'text'
      }, {
        key: 'itControllerNo',
        header: 'IT Controller S/N',
        type: 'text'
      }, {
        key: 'invoiceNo',
        header: 'Invoice',
        type: 'text'
      }, {
        key: 'invoiceDate',
        header: 'Invoice Date',
        type: 'center',
        width: 14
      }, {
        key: 'exportEntry',
        header: 'Export Entry',
        type: 'text'
      }, {
        key: 'importLicenseNo',
        header: 'Import License',
        type: 'text'
      }, {
        key: 'exportLicenseNo',
        header: 'Export License',
        type: 'text'
      }, {
        key: 'country',
        header: 'Country',
        type: 'center',
        width: 14
      }, {
        key: 'remark',
        header: 'Remark',
        type: 'text'
      }];
      const dash2 = v => v && String(v).trim() !== '' ? String(v) : '—';
      const sheets = countryNames.map(country => {
        const list = [...groups.get(country)].sort((a, b) => {
          const ta = a.IssueDate ? new Date(a.IssueDate).getTime() : Infinity;
          const tb = b.IssueDate ? new Date(b.IssueDate).getTime() : Infinity;
          return ta - tb;
        });
        const extra = collectExtraColumns(list);
        const extraColumns = extra.spread.map((label, idx) => ({
          key: `x${idx}`,
          header: label,
          type: 'text'
        }));
        if (extra.overflow.length) {
          extraColumns.push({
            key: 'xOverflow',
            header: `คอลัมน์เพิ่ม (อีก ${extra.overflow.length} คอลัมน์)`,
            type: 'text',
            width: 40
          });
        }
        return {
          sheetName: countryLabel(country),
          columns: [...baseColumns, ...extraColumns],
          rows: list.map((r, i) => {
            const exp = computeExportExpiry(r);
            const extraValues = parseExtraJson(r.extra_json);
            const done = isLicenseCompleted(r);
            const row = {
              __danger: !done && (exp.status === EXPIRY_STATUS.EXPIRED || exp.leadStatus === LEAD_STATUS.OVERDUE),
              item: i + 1,
              assemblyDate: r.AssemblyDate ? formatThaiDate(r.AssemblyDate) : '—',
              issueDate: r.IssueDate ? formatThaiDate(r.IssueDate) : '—',
              expiryDate: exp.hasDate ? formatThaiDate(exp.expiryDate) : '—',
              leadTimeDate: exp.hasDate ? formatThaiDate(exp.leadDate) : '—',
              completedDate: done && r.CompletedAt ? formatThaiDate(r.CompletedAt) : '—',
              machineNo: dash2(r.MachineNo),
              itControllerNo: dash2(r.ITControllerNo),
              invoiceNo: dash2(r.InvoiceNo),
              invoiceDate: r.InvoiceDate ? formatThaiDate(r.InvoiceDate) : '—',
              exportEntry: dash2(r.ExportEntry),
              importLicenseNo: dash2(r.ImportLicenseNo),
              exportLicenseNo: dash2(r.ExportLicenseNo),
              country: countryLabel(country),
              remark: dash2(r.Remark)
            };
            extra.spread.forEach((label, idx) => {
              row[`x${idx}`] = extraValues[label] ?? '';
            });
            if (extra.overflow.length) {
              row.xOverflow = extra.overflow.filter(label => extraValues[label]).map(label => `${label}: ${extraValues[label]}`).join('\n');
            }
            return row;
          })
        };
      });
      const blob = buildStyledXlsxWorkbookBlob({
        sheets
      });
      downloadBlob(blob, `ExportLicense-by-country-${periodTag}.xlsx`);
      const scope = periodMode === 'all' ? '' : ` — ช่วง ${periodLabel}`;
      toastSuccess(`Export สำเร็จ — ${countryNames.length} ประเทศ (${list.length} รายการ)${scope}`);
    } catch (err) {
      toastError(err.message || 'Export ไม่สำเร็จ');
    } finally {
      setExportingXlsx(false);
    }
  }
  async function handlePreview() {
    if (!file) {
      setMsg({
        error: 'กรุณาเลือกไฟล์ก่อนตรวจสอบ'
      });
      return;
    }
    setPreviewing(true);
    setPreviewData(null);
    try {
      const data = await previewExportLicense(file);
      setPreviewData(data);
    } catch (err) {
      setMsg({
        error: err.message || 'ตรวจสอบไฟล์ไม่สำเร็จ'
      });
    } finally {
      setPreviewing(false);
    }
  }
  const loadSeq = useRef(0);
  async function load(silent = false) {
    const seq = ++loadSeq.current;
    if (!silent) setLoading(true);
    try {
      const data = await getExportLicense();
      if (seq === loadSeq.current) setRows(data);
    } catch (err) {
      if (!silent) toastError(err.message || 'โหลดบัญชีใบอนุญาตส่งออกไม่สำเร็จ');
    } finally {
      if (!silent && seq === loadSeq.current) setLoading(false);
    }
  }
  useEffect(() => {
    load();
  }, []);
  useLiveRefresh(() => load(true));
  async function saveExportField(row, key, value) {
    const updated = await updateExportLicense(row.ID, {
      [key]: value
    });
    setRows(list => (list || []).map(r => r.ID === row.ID ? {
      ...r,
      ...updated
    } : r));
  }
  async function handleRenewSelectedExport() {
    const licenseNo = licenseFilter;
    if (!licenseNo || licenseNo === 'all') {
      toastError('กรุณาเลือกใบอนุญาตส่งออกที่ต้องการต่ออายุก่อน');
      return;
    }
    const days = await promptRenewDays({
      title: `ต่ออายุใบอนุญาตส่งออก ${licenseNo}`,
      html: '',
      defaultDays: 180
    });
    if (!days) return;
    try {
      const res = await renewExportLicense(licenseNo, '', days);
      await load();
      const newExp = res?.newExpiry ? formatThaiDate(new Date(res.newExpiry)) : '';
      toastSuccess(`ต่ออายุใบอนุญาตส่งออก ${licenseNo} อีก ${days} วันแล้ว${newExp ? ` — หมดอายุ ${newExp}` : ''}`);
    } catch (err) {
      toastError(err.message || 'ต่ออายุไม่สำเร็จ');
    }
  }
  useEffect(() => {
    setPage(1);
  }, [search, licenseFilter, countryFilter, expiryFilter, pageSize, periodMode, periodAnchor]);
  useEffect(() => {
    const lic = (params?.focusLicenseNo || '').trim();
    const itc = (params?.focusITC || '').trim();
    if (!lic && !itc) return;
    setExpiryFilter('all');
    setCountryFilter(ALL_COUNTRIES);
    setPeriodMode('all');
    setPeriodAnchor('');
    setLicenseFilter('all');
    setSearch(lic || itc);
  }, [params?.focusITC, params?.focusLicenseNo, params?.focusTs]);
  useEffect(() => {
    const lic = (params?.focusLicenseNo || '').trim();
    if (!lic) return;
    if (rows.some(r => (r.ExportLicenseNo || '') === lic)) {
      setLicenseFilter(lic);
      setSearch('');
    }
  }, [rows, params?.focusLicenseNo, params?.focusTs]);
  async function handleUpload() {
    if (!file) {
      setMsg({
        error: 'กรุณาเลือกไฟล์ Excel หรือ CSV ก่อน'
      });
      return;
    }
    setUploading(true);
    setMsg(null);
    try {
      const summary = previewData?.summary || (await previewExportLicense(file))?.summary;
      if (!(await confirmUploadDeletes(summary, file.name))) {
        return;
      }
      const r = await uploadExportLicense(file);
      setMsg({
        success: [`เพิ่มใหม่ ${r.imported ?? 0}`, `อัปเดต ${r.updated ?? 0}`, ...(r.unchanged ? [`เหมือนเดิม ${r.unchanged}`] : []), ...syncResultParts(r), `ข้าม ${r.skipped ?? 0}`].join(' · '),
        problems: r.problems || []
      });
      setFile(null);
      setPreviewData(null);
      await load();
    } catch (err) {
      setMsg({
        error: err.message || 'อัปโหลดไม่สำเร็จ'
      });
    } finally {
      setUploading(false);
    }
  }
  async function handleDelete(row) {
    const ok = await confirmDelete({
      text: `ลบ IT Controller S/N ${row.ITControllerNo || '—'} ออกจากบัญชี?`
    });
    if (!ok) return;
    try {
      await deleteExportLicense(row.ID);
      await load();
      toastSuccess(`ลบ ${row.ITControllerNo || ''} แล้ว`);
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    }
  }
  async function handleClearAll() {
    const ok = await confirmDelete({
      text: 'ลบบัญชีใบอนุญาตส่งออกทั้งหมด? กู้คืนไม่ได้',
      confirmText: 'ลบทั้งหมด'
    });
    if (!ok) return;
    try {
      await clearExportLicense();
      await load();
      toastSuccess('ลบบัญชีใบอนุญาตส่งออกทั้งหมดแล้ว');
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    }
  }
  async function handleClearSelectedExportLicense() {
    const licenseNo = licenseFilter;
    if (!licenseNo || licenseNo === 'all') return;
    const ok = await confirmDelete({
      text: `ลบใบอนุญาตส่งออก ${licenseNo} ออกจากระบบทั้งใบ? กู้คืนไม่ได้`,
      confirmText: 'ลบทั้งใบ'
    });
    if (!ok) return;
    try {
      const res = await clearExportLicense(licenseNo);
      setLicenseFilter('all');
      await load();
      toastSuccess(`ลบใบอนุญาตส่งออก ${licenseNo} แล้ว (${res?.deleted ?? 0} รายการ)`);
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    }
  }
  const filtered = useMemo(() => {
    let list = rows;
    if (licenseFilter !== 'all') {
      list = list.filter(r => (r.ExportLicenseNo || '') === licenseFilter);
    }
    if (countryFilter !== ALL_COUNTRIES) {
      list = list.filter(r => countryKey(countryOf(r)) === countryFilter);
    }
    const term = search.trim().toLowerCase();
    if (term) {
      list = list.filter(r => [r.MachineNo, r.ITControllerNo, r.InvoiceNo, r.ExportEntry, r.ImportLicenseNo, r.ExportLicenseNo, countryOf(r)].filter(Boolean).some(v => String(v).toLowerCase().includes(term)));
    }
    if (periodMode !== 'all') {
      list = list.filter(r => r.AssemblyDate && inPeriod(r.AssemblyDate, periodMode, periodAnchor));
    }
    if (expiryFilter === COMPLETED_FILTER) {
      list = list.filter(isLicenseCompleted);
    } else if (expiryFilter !== 'all') {
      list = list.filter(r => {
        if (isLicenseCompleted(r)) return false;
        const exp = computeExportExpiry(r);
        if (expiryFilter === LEAD_STATUS.OVERDUE || expiryFilter === LEAD_STATUS.DUE) {
          return exp.leadStatus === expiryFilter;
        }
        return exp.status === expiryFilter;
      });
    }
    return list;
  }, [rows, licenseFilter, countryFilter, expiryFilter, search, countryByITC, periodMode, periodAnchor]);
  const countryOptions = useMemo(() => buildCountryOptions(rows.map(countryOf)), [rows, countryByITC]);
  const filterActive = licenseFilter !== 'all' || countryFilter !== ALL_COUNTRIES || expiryFilter !== 'all' || search.trim() !== '' || periodMode !== 'all';
  useEffect(() => {
    if (countryFilter !== ALL_COUNTRIES && !countryOptions.some(o => o.value === countryFilter)) {
      setCountryFilter(ALL_COUNTRIES);
    }
  }, [countryOptions, countryFilter]);
  const asmDateBounds = useMemo(() => {
    let min = null;
    let max = null;
    rows.forEach(r => {
      if (!r.AssemblyDate) return;
      const d = new Date(r.AssemblyDate);
      if (Number.isNaN(d.getTime())) return;
      const ymd = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
      if (min === null || ymd < min) min = ymd;
      if (max === null || ymd > max) max = ymd;
    });
    return {
      min,
      max
    };
  }, [rows]);
  function handlePeriodModeChange(next) {
    setPeriodMode(next);
    if (next !== 'all' && !periodAnchor) {
      const today = new Date();
      const todayYMD = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
      setPeriodAnchor(asmDateBounds.max || todayYMD);
    }
  }
  const periodLabel = periodMode === 'all' ? 'ทั้งหมด' : periodRangeLabel(periodMode, periodAnchor);
  const periodTag = periodFileTag(periodMode, periodAnchor);
  const licenseOptions = useMemo(() => {
    const stat = new Map();
    rows.forEach(r => {
      const key = r.ExportLicenseNo;
      if (!key) return;
      const cur = stat.get(key) || {
        total: 0,
        done: 0
      };
      cur.total += 1;
      if (isLicenseCompleted(r)) cur.done += 1;
      stat.set(key, cur);
    });
    const list = Array.from(stat.keys()).sort((a, b) => a.localeCompare(b));
    return [{
      value: 'all',
      label: 'ทุกใบอนุญาต'
    }, ...list.map(m => {
      const st = stat.get(m);
      return {
        value: m,
        label: licenseOptionLabel(m, st.total, st.done),
        suffix: st.total > 0 && st.done >= st.total ? <CompletedOptionIcon /> : null
      };
    })];
  }, [rows]);
  const expiryOptions = useMemo(() => [{
    value: 'all',
    label: 'ทุกสถานะ'
  }, {
    value: LEAD_STATUS.OVERDUE,
    label: `Lead time · ${LEAD_STATUS_LABEL[LEAD_STATUS.OVERDUE]}`
  }, {
    value: LEAD_STATUS.DUE,
    label: `Lead time · ${LEAD_STATUS_LABEL[LEAD_STATUS.DUE]}`
  }, {
    value: EXPIRY_STATUS.NO_DATE,
    label: STATUS_LABEL[EXPIRY_STATUS.NO_DATE]
  }, {
    value: EXPIRY_STATUS.EXPIRING,
    label: STATUS_LABEL[EXPIRY_STATUS.EXPIRING]
  }, {
    value: EXPIRY_STATUS.EXPIRED,
    label: STATUS_LABEL[EXPIRY_STATUS.EXPIRED]
  }, {
    value: EXPIRY_STATUS.VALID,
    label: STATUS_LABEL[EXPIRY_STATUS.VALID]
  }, {
    value: COMPLETED_FILTER,
    label: COMPLETED_LABEL
  }], []);
  const currentLicenseRows = useMemo(() => {
    if (licenseFilter === 'all') return [];
    return rows.filter(r => (r.ExportLicenseNo || '') === licenseFilter);
  }, [rows, licenseFilter]);
  const currentLicenseInvoices = useMemo(() => {
    const set = new Set(currentLicenseRows.map(r => r.InvoiceNo).filter(Boolean));
    return Array.from(set).sort((a, b) => a.localeCompare(b));
  }, [currentLicenseRows]);
  const currentLicenseEntries = useMemo(() => {
    const set = new Set(currentLicenseRows.map(r => (r.ExportEntry || '').trim()).filter(Boolean));
    return Array.from(set).sort((a, b) => a.localeCompare(b));
  }, [currentLicenseRows]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);

  const {
    selected,
    toggleOne,
    replaceWith: replaceSelection,
    clear: clearSelection,
    toggleAll,
    invert: invertSelection,
    allSelected,
    someSelected
  } = useRowSelection(filtered);
  const selectedRows = useMemo(() => filtered.filter(r => selected.has(r.ID)), [filtered, selected]);
  const licenseValues = useMemo(() => licenseOptions.filter(o => o.value !== 'all').map(o => o.value), [licenseOptions]);
  const {
    checked: checkedLicenses,
    toggle: toggleLicenseCheck,
    setAll: setAllLicenseChecks,
    clear: clearLicenseChecks
  } = useCheckedSet(licenseValues);
  const checkedLicenseRows = useMemo(() => checkedLicenses.size === 0 ? [] : rows.filter(r => checkedLicenses.has(r.ExportLicenseNo || '')), [rows, checkedLicenses]);
  const checkedLicenseDone = useMemo(() => checkedLicenseRows.filter(isLicenseCompleted).length, [checkedLicenseRows]);

  async function completeCheckedLicenses(completed) {
    const targets = completed ? checkedLicenseRows.filter(r => !isLicenseCompleted(r)) : checkedLicenseRows.filter(isLicenseCompleted);
    if (targets.length === 0) return;
    const n = checkedLicenses.size;
    const ok = await confirmComplete({
      title: completed ? `ทำเครื่องหมายเสร็จสิ้น ${fmtCount(n)} ใบอนุญาต?` : `ยกเลิกสถานะเสร็จสิ้น ${fmtCount(n)} ใบอนุญาต?`,
      html: `${fmtCount(targets.length)} รายการ`,
      danger: !completed
    });
    if (!ok) return;
    setCompleting(true);
    try {
      await setExportLicenseComplete({
        ids: targets.map(r => r.ID),
        completed
      });
      clearLicenseChecks();
      await load();
      toastSuccess(completed ? `ปิดงาน ${fmtCount(n)} ใบอนุญาตแล้ว (${fmtCount(targets.length)} รายการ) — หยุดนับวันหมดอายุ` : `ยกเลิกสถานะเสร็จสิ้น ${fmtCount(n)} ใบอนุญาตแล้ว`);
    } catch (err) {
      toastError(err.message || 'อัปเดตสถานะไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }

  async function deleteCheckedLicenses() {
    // กันไว้: clearExportLicense('') = ลบทั้งหมด จึงส่งเฉพาะเลขที่ไม่ว่าง
    const licenses = [...checkedLicenses].filter(v => String(v || '').trim() !== '');
    if (licenses.length === 0) return;
    const names = licenses.slice(0, 5).join(', ') + (licenses.length > 5 ? ` และอีก ${licenses.length - 5} ใบ` : '');
    const ok = await confirmDelete({
      title: `ลบ ${fmtCount(licenses.length)} ใบอนุญาตส่งออก?`,
      text: `${names} (${fmtCount(checkedLicenseRows.length)} รายการ) — ลบแล้วกู้คืนไม่ได้`,
      confirmText: `ลบ ${fmtCount(licenses.length)} ใบ`
    });
    if (!ok) return;
    setCompleting(true);
    let deleted = 0;
    let failed = 0;
    try {
      for (const lic of licenses) {
        try {
          const res = await clearExportLicense(lic);
          deleted += res?.deleted ?? 0;
        } catch {
          failed += 1;
        }
      }
      if (licenses.includes(licenseFilter)) setLicenseFilter('all');
      clearLicenseChecks();
      clearSelection();
      await load();
      if (failed > 0) toastError(`ลบไม่สำเร็จ ${fmtCount(failed)} ใบ`);else toastSuccess(`ลบ ${fmtCount(licenses.length)} ใบอนุญาตแล้ว (${fmtCount(deleted)} รายการ)`);
    } finally {
      setCompleting(false);
    }
  }

  async function applyBulkDelete() {
    const targets = selectedRows;
    if (targets.length === 0) return;
    const ok = await confirmDelete({
      title: `ลบ ${fmtCount(targets.length)} รายการที่เลือก?`,
      text: 'ลบออกจากระบบแล้วกู้คืนไม่ได้',
      confirmText: `ลบ ${fmtCount(targets.length)} รายการ`
    });
    if (!ok) return;
    setCompleting(true);
    try {
      const res = await bulkDeleteExportLicense(targets.map(r => r.ID));
      clearSelection();
      await load();
      toastSuccess(`ลบแล้ว ${fmtCount(res?.deleted ?? targets.length)} รายการ`);
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }
  const completedCount = useMemo(() => rows.filter(isLicenseCompleted).length, [rows]);

  const exportCounts = useMemo(() => ({
    total: rows.length,
    licenses: new Set(rows.map(r => (r.ExportLicenseNo || '').trim()).filter(Boolean)).size,
    entries: new Set(rows.map(r => (r.ExportEntry || '').trim()).filter(Boolean)).size,
    completed: completedCount
  }), [rows, completedCount]);

  const exportExpiryCounts = useMemo(() => {
    const worst = new Map();
    for (const row of rows) {
      if (isLicenseCompleted(row)) continue;
      const key = (row.ExportLicenseNo || '').trim();
      if (!key) continue;
      const exp = computeExportLicenseDates(row);
      if (exp.status !== EXPIRY_STATUS.EXPIRED && exp.status !== EXPIRY_STATUS.EXPIRING) continue;
      if (worst.get(key) === EXPIRY_STATUS.EXPIRED) continue;
      worst.set(key, exp.status);
    }
    let expiring = 0;
    let expired = 0;
    for (const status of worst.values()) {
      if (status === EXPIRY_STATUS.EXPIRED) expired++;else expiring++;
    }
    return {
      expiring,
      expired
    };
  }, [rows]);
  const currentLicenseCompletedAll = currentLicenseRows.length > 0 && currentLicenseRows.every(isLicenseCompleted);
  const [refsOpen, setRefsOpen] = useState(false);
  useEffect(() => {
    setRefsOpen(false);
  }, [licenseFilter]);
  const hasMoreRefs = currentLicenseInvoices.length > 1 || currentLicenseEntries.length > 1;
  const refsTitle = [`Invoice (${currentLicenseInvoices.length}): ${currentLicenseInvoices.join(', ') || '—'}`, `ใบขนสินค้าขาออก (${currentLicenseEntries.length}): ${currentLicenseEntries.join(', ') || '—'}`].join('\n');

  async function applyComplete(completed) {
    const targets = completed ? selectedRows.filter(r => !isLicenseCompleted(r)) : selectedRows.filter(isLicenseCompleted);
    if (targets.length === 0) return;
    const ok = await confirmComplete({
      title: completed ? `ต้องการทำเครื่องหมายเสร็จสิ้น ${fmtCount(targets.length)} รายการ` : `ต้องการยกเลิกสถานะเสร็จสิ้น ${fmtCount(targets.length)} รายการ`,
      danger: !completed
    });
    if (!ok) return;
    setCompleting(true);
    try {
      await setExportLicenseComplete({
        ids: targets.map(r => r.ID),
        completed
      });
      clearSelection();
      await load();
      toastSuccess(completed ? `ทำเครื่องหมายเสร็จสิ้น ${fmtCount(targets.length)} รายการแล้ว — หยุดนับวันหมดอายุและ Lead time` : `ยกเลิกสถานะเสร็จสิ้น ${fmtCount(targets.length)} รายการแล้ว`);
    } catch (err) {
      toastError(err.message || 'อัปเดตสถานะไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }

  async function handleCompleteSelectedLicense(completed) {
    const licenseNo = licenseFilter;
    if (!licenseNo || licenseNo === 'all') return;
    const ok = await confirmComplete({
      title: completed ? `ต้องการทำเครื่องหมายเสร็จสิ้นทั้งใบ ${licenseNo}` : `ต้องการยกเลิกสถานะเสร็จสิ้นทั้งใบ ${licenseNo}`,
      danger: !completed
    });
    if (!ok) return;
    setCompleting(true);
    try {
      const res = await setExportLicenseComplete({
        exportLicenseNo: licenseNo,
        completed
      });
      clearSelection();
      await load();
      toastSuccess(completed ? `ปิดงาน ${licenseNo} แล้ว (${res?.updated ?? 0} รายการ) — หยุดนับวันหมดอายุ` : `ยกเลิกสถานะเสร็จสิ้น ${licenseNo} แล้ว (${res?.updated ?? 0} รายการ)`);
    } catch (err) {
      toastError(err.message || 'อัปเดตสถานะไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }

  async function handleToggleRowComplete(row) {
    const completed = !isLicenseCompleted(row);
    setCompleting(true);
    try {
      await setExportLicenseComplete({
        ids: [row.ID],
        completed
      });
      setTraceRow(null);
      await load();
      toastSuccess(completed ? `${row.MachineNo || row.ITControllerNo || 'รายการนี้'} เสร็จสิ้นแล้ว — หยุดนับวันหมดอายุ` : `ยกเลิกสถานะเสร็จสิ้นของ ${row.MachineNo || row.ITControllerNo || 'รายการนี้'} แล้ว`);
    } catch (err) {
      toastError(err.message || 'อัปเดตสถานะไม่สำเร็จ');
    } finally {
      setCompleting(false);
    }
  }

  return <>
      <div className="wh-upload-card">
        <div className="fdz-row">
          <FileDropZone file={file} onSelect={f => {
          setFile(f);
          setMsg(null);
          setPreviewData(null);
        }} accept=".xlsx,.xls,.csv" label="อัปโหลดบัญชีใบอนุญาตส่งออก" disabled={uploading} />
          <button className="wh-modal-cancel" onClick={handlePreview} disabled={previewing || uploading || !file}>
            {previewing ? 'กำลังตรวจสอบ...' : 'ตรวจสอบก่อนอัปโหลด'}
          </button>
          <button className="wh-issue-btn" onClick={handleUpload} disabled={uploading || !file}>
            {uploading ? 'กำลังอัปโหลด...' : 'อัปโหลด'}
          </button>
        </div>
        {previewData && (previewData.summary ? <ChangePreview result={previewData} /> : <PreviewResult result={previewData} />)}
        {msg?.success && <p className="upload-card-msg upload-card-msg-ok wh-upload-msg">{msg.success}</p>}
        {msg?.error && <p className="upload-card-msg upload-card-msg-err wh-upload-msg">{msg.error}</p>}
        {msg?.problems?.length > 0 && <ul className="il-problem-list">
            {msg.problems.map((p, i) => <li key={i}>{p}</li>)}
          </ul>}
      </div>

      <div className="dash-stats-row wh-stats-row il-stats-row-5">
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <StatLabel parts={['เครื่องในบัญชี', 'ทั้งหมด']} />
            <span className="dash-stat-icon dash-icon-blue">
              <Squares2X2Icon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{exportCounts.total}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <StatLabel parts={['ใบอนุญาต', 'นำออก']} />
            <span className="dash-stat-icon dash-icon-red">
              <DocumentTextIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{exportCounts.licenses}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <StatLabel parts={['ใบขนสินค้า', 'ขาออก']} />
            <span className="dash-stat-icon dash-icon-yellow">
              <TruckIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{exportCounts.entries}</div>
        </div>
        <div className="dash-stat-card il-stat-expiry">
          <div className="dash-stat-label">
            <StatLabel parts={['อายุ', 'ใบอนุญาต']} />
            <span className="dash-stat-icon dash-icon-orange">
              <ClockIcon className="size-4" />
            </span>
          </div>
          <div className="il-stat-expiry-split">
            <div className="il-stat-expiry-part il-stat-expiring">
              <div className="dash-stat-value">{exportExpiryCounts.expiring}</div>
              <div className="dash-stat-note">ใกล้หมดอายุ</div>
            </div>
            <span className="il-stat-expiry-divider" aria-hidden="true" />
            <div className="il-stat-expiry-part il-stat-expired">
              <div className="dash-stat-value">{exportExpiryCounts.expired}</div>
              <div className="dash-stat-note">หมดอายุแล้ว</div>
            </div>
          </div>
        </div>
        <div className="dash-stat-card il-stat-complete">
          <div className="dash-stat-label">
            <StatLabel parts={['เสร็จสิ้นแล้ว']} />
            <span className="dash-stat-icon dash-icon-green">
              <CheckBadgeIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{exportCounts.completed}</div>
        </div>
      </div>

      <div className="il-export-filter-card">
        <PeriodRangePicker mode={periodMode} onModeChange={handlePeriodModeChange} anchor={periodAnchor} onAnchorChange={setPeriodAnchor} min={asmDateBounds.min} max={asmDateBounds.max} label="เลือกช่วงวันที่สำหรับ Export แยกประเทศ" countLabel={`${filtered.length} รายการ`} onClear={() => {
        setPeriodMode('all');
        setPeriodAnchor('');
      }} />
      </div>

      {rows.length > 0 && <div className="il-lot-filter">
          <label className="il-lot-filter-label">ใบอนุญาตส่งออก</label>
          <div className="il-lot-filter-select">
            <LicenseCheckSelect value={licenseFilter} onChange={setLicenseFilter} options={licenseOptions} allValue="all" checked={checkedLicenses} onToggleCheck={toggleLicenseCheck} onCheckAll={setAllLicenseChecks} />
          </div>
        </div>}

      <LicenseActionBar licenseCount={checkedLicenses.size} machineCount={checkedLicenseRows.length} openCount={checkedLicenseRows.length - checkedLicenseDone} doneCount={checkedLicenseDone} busy={completing} onComplete={() => completeCheckedLicenses(true)} onUncomplete={() => completeCheckedLicenses(false)} onDelete={deleteCheckedLicenses} onClear={clearLicenseChecks} />

      {licenseFilter !== 'all' && <div className={'wh-so-active-bar il-license-bar' + (refsOpen ? ' il-license-bar-open' : '')}>
          <div className="il-lot-info">
            <div className="il-lot-info-text">
              <span className="wh-so-active-label">ใบอนุญาตส่งออก</span>
              <h3 className="wh-so-active-name">{licenseFilter || '(ไม่มีเลขใบอนุญาต)'}</h3>
              <div className="wh-subtitle il-lot-refs">
                <span className="il-lot-refs-text" title={refsTitle}>
                  <LicenseRefSummary label="Invoice" values={currentLicenseInvoices} />
                  <span className="il-ref-sep" aria-hidden="true">·</span>
                  <LicenseRefSummary label="ใบขนสินค้าขาออก" values={currentLicenseEntries} />
                </span>
                <span className="il-ref-count">
                  <span className="il-ref-sep" aria-hidden="true">·</span>
                  {currentLicenseRows.length} เครื่อง
                </span>
                {hasMoreRefs && <button type="button" className={'il-ref-toggle' + (refsOpen ? ' il-ref-toggle-open' : '')} onClick={() => setRefsOpen(v => !v)} aria-expanded={refsOpen}>
                    {refsOpen ? 'ย่อ' : 'ดูทั้งหมด'}
                    <ChevronDownIcon className="il-ref-toggle-icon" aria-hidden="true" />
                  </button>}
              </div>
              {refsOpen && <div className="il-lot-refs-detail">
                  <LicenseRefList label="Invoice" values={currentLicenseInvoices} />
                  <LicenseRefList label="ใบขนสินค้าขาออก" values={currentLicenseEntries} />
                </div>}
            </div>
            {currentLicenseCompletedAll && <span className="il-complete-stamp" title="ปิดงานทั้งใบแล้ว">
                <CheckBadgeSolidIcon className="il-complete-stamp-icon" aria-hidden="true" />
                เสร็จสิ้นแล้ว
              </span>}
          </div>
          <div className="il-lot-actions">
            <button className={currentLicenseCompletedAll ? 'il-uncomplete-btn' : 'il-complete-btn'} onClick={() => handleCompleteSelectedLicense(!currentLicenseCompletedAll)} disabled={completing} title={currentLicenseCompletedAll ? 'กลับมานับวันหมดอายุและแจ้งเตือนใบนี้ตามปกติ' : 'ปิดงานทั้งใบ แล้วหยุดนับวันหมดอายุ'}>
              {currentLicenseCompletedAll ? <>
                  <ArrowPathIcon className="size-4" />
                  ยกเลิกเสร็จสิ้นทั้งใบ
                </> : <>
                  <CheckBadgeIcon className="size-4" />
                  ยืนยันเสร็จสิ้น
                </>}
            </button>
            <button className="wh-issue-btn il-renew-btn" onClick={handleRenewSelectedExport}>
              ต่ออายุ
            </button>
            <button className="wh-modal-cancel" onClick={handleClearSelectedExportLicense}>
              ลบทั้งใบ
            </button>
          </div>
        </div>}

      <div className="tsf-history-toolbar">
        <div className="tsf-history-pagesize">
          <div className="wh-pagesize-select">
            <SelectField value={pageSize} onChange={setPageSize} options={[{
            value: 10,
            label: '10'
          }, {
            value: 25,
            label: '25'
          }, {
            value: 50,
            label: '50'
          }, {
            value: 100,
            label: '100'
          }]} />
          </div>
          entries per page
        </div>
        <div className="il-filter-search-group il-filter-search-group-compact">
          <div className="wh-pagesize-select il-model-filter">
            <SelectField value={countryFilter} onChange={setCountryFilter} options={countryOptions} />
          </div>
          <div className="wh-pagesize-select il-model-filter">
            <SelectField value={expiryFilter} onChange={setExpiryFilter} options={expiryOptions} />
          </div>
          <input className="wh-search" type="text" placeholder="ค้นหา Machine No / IT Controller / Invoice / License / ประเทศ" value={search} onChange={e => setSearch(e.target.value)} />
          <button className="wh-issue-btn" onClick={handleExportByCountry} disabled={exportingXlsx || rows.length === 0} title={`ดาวน์โหลด Excel แยกชีตตามประเทศปลายทาง — ช่วง ${periodLabel}`}>
            {exportingXlsx ? 'กำลัง Export...' : 'Export Excel'}
          </button>
          {rows.length > 0 && <button className="wh-btn-danger" onClick={handleClearAll}>
              ลบทุกใบอนุญาต
            </button>}
        </div>
      </div>

      <SelectionBar selectedRows={selectedRows} busy={completing} onComplete={() => applyComplete(true)} onUncomplete={() => applyComplete(false)} onDelete={applyBulkDelete} onInvert={invertSelection} onClear={clearSelection} allSelected={allSelected} filterActive={filterActive} />

      <div className="wh-table-card">
        <table className="wh-table il-table-selectable">
          <thead>
            <tr>
              <th className="il-check-th">
                <SelectCheckbox checked={allSelected} indeterminate={someSelected} onChange={toggleAll} label={`เลือกทุกรายการ (${fmtCount(filtered.length)} รายการ)`} title={allSelected ? 'ยกเลิกการเลือกทั้งหมด' : `เลือกทุกรายการ ${fmtCount(filtered.length)} รายการ`} disabled={filtered.length === 0} />
              </th>
              <th>Item</th>
              <th>Date Ass'y</th>
              <th>Machine No</th>
              <th>IT Controller S/N</th>
              <th>Country</th>
              <th>Invoice</th>
              <th>Export Entry</th>
              <th>Import License</th>
              <th>Export License</th>
              <th>วันที่นำออกใบอนุญาต</th>
              <th>หมดอายุ (1 เดือน)</th>
              <th>Lead time ({EXPORT_LICENSE_LEAD_DAYS} วัน)</th>
              <th>Remark</th>
              <th>คอลัมน์เพิ่ม</th>
              <th>สถานะการประกอบ</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={17} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>}
            {!loading && paged.map((row, i) => {
            const save = key => v => saveExportField(row, key, v);
            return <tr key={row.ID} className={(isLicenseCompleted(row) ? 'il-row-complete' : '') + (selected.has(row.ID) ? ' il-row-selected' : '')}>
                  <td className="il-check-td" data-label="เลือก">
                    <SelectCheckbox checked={selected.has(row.ID)} onChange={(_, shift) => toggleOne(row.ID, shift)} label={`เลือก ${row.MachineNo || row.ITControllerNo || 'รายการนี้'}`} title="Shift+คลิก เพื่อเลือกเป็นช่วง" />
                  </td>
                  <td className="wh-cell-head" data-label="Item">
                    {(page - 1) * pageSize + i + 1}
                  </td>
                  <td data-label="Date Ass'y">
                    <EditableCell readOnly type="date" label="Date Ass'y" value={row.AssemblyDate} display={formatThaiDate(row.AssemblyDate)} onSave={save('AssemblyDate')} />
                  </td>
                  <td className="il-mono wh-cell-head" data-label="Machine No">
                    <EditableCell readOnly mono label="Machine No" value={row.MachineNo} display={<strong>{row.MachineNo || '—'}</strong>} onSave={save('MachineNo')} />
                  </td>
                  <td className="il-mono" data-label="IT Controller S/N">
                    <EditableCell readOnly mono label="IT Controller S/N" value={row.ITControllerNo} onSave={save('ITControllerNo')} />
                  </td>
                  <td data-label="Country">
                    <EditableCell readOnly label="Country" value={row.Country} display={countryOf(row) || <span className="il-no-country">{NO_COUNTRY_LABEL}</span>} onSave={save('Country')} />
                  </td>
                  <td data-label="Invoice">
                    <EditableCell readOnly mono label="Invoice" value={row.InvoiceNo} onSave={save('InvoiceNo')} />
                    {row.InvoiceDate && <div className="il-invoice-date">
                        {formatThaiDate(row.InvoiceDate)}
                      </div>}
                  </td>
                  <td className="il-mono" data-label="Export Entry">
                    <EditableCell readOnly mono label="Export Entry" value={row.ExportEntry} onSave={save('ExportEntry')} />
                  </td>
                  <td className="il-mono" data-label="Import License">
                    <EditableCell readOnly mono label="Import License" value={row.ImportLicenseNo} onSave={save('ImportLicenseNo')} />
                  </td>
                  <td className="il-mono" data-label="Export License">
                    <span className="il-license-cell">
                      <CompleteFlag show={isLicenseCompleted(row)} />
                      <EditableCell readOnly mono label="Export License" value={row.ExportLicenseNo} onSave={save('ExportLicenseNo')} />
                    </span>
                  </td>
                  <td data-label="วันที่นำออกใบอนุญาต">
                    <EditableCell readOnly type="date" label="วันที่นำออกใบอนุญาต" value={row.IssueDate} display={formatThaiDate(row.IssueDate)} onSave={save('IssueDate')} />
                  </td>
                  <td data-label="หมดอายุ (1 เดือน)">
                    <ExportOneMonthExpiryCell row={row} />
                  </td>
                  <td data-label={`Lead time (${EXPORT_LICENSE_LEAD_DAYS} วัน)`}>
                    <ExportLeadTimeCell row={row} />
                  </td>
                  <td data-label="Remark">
                    <EditableCell readOnly label="Remark" value={row.Remark} onSave={save('Remark')} />
                  </td>
                  <td data-label="คอลัมน์เพิ่ม">
                    <ExtraColumnsCell json={row.extra_json} />
                  </td>
                  <td data-label="สถานะการประกอบ">
                    <AssemblyStatusCell row={row} />
                  </td>
                  <td className="wh-cell-action">
                    <div className="il-row-actions">
                      <button className="wh-modal-cancel" onClick={() => setTraceRow(row)}>
                        รายละเอียด
                      </button>
                      <button className="wh-btn-danger" onClick={() => handleDelete(row)}>
                        ลบ
                      </button>
                    </div>
                  </td>
                </tr>;
          })}
            {!loading && paged.length === 0 && <tr>
                <td colSpan={17} className="wh-empty-cell">
                  ยังไม่มีข้อมูล
                </td>
              </tr>}
          </tbody>
        </table>
      </div>

      {traceRow && <ExportTraceModal row={traceRow} country={countryOf(traceRow)} busy={completing} onToggleComplete={handleToggleRowComplete} onClose={() => setTraceRow(null)} />}

      {!loading && filtered.length > pageSize && <div className="tsf-pagination">
          <span className="wh-subtitle" style={{
        fontSize: 13
      }}>
            Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, filtered.length)} of{' '}
            {filtered.length} entries
          </span>
          <div className="tsf-pagination-buttons">
            <button className="wh-modal-cancel" onClick={() => setPage(1)} disabled={page === 1} title="หน้าแรก" aria-label="หน้าแรก">
              <ChevronDoubleLeftIcon className="size-4" />
            </button>
            <button className="wh-modal-cancel" onClick={() => setPage(Math.max(1, page - 1))} disabled={page === 1} title="หน้าก่อนหน้า" aria-label="หน้าก่อนหน้า">
              <ChevronLeftIcon className="size-4" />
            </button>
            <span className="tsf-pagination-current">
              {page} / {totalPages}
            </span>
            <button className="wh-modal-cancel" onClick={() => setPage(Math.min(totalPages, page + 1))} disabled={page === totalPages} title="หน้าถัดไป" aria-label="หน้าถัดไป">
              <ChevronRightIcon className="size-4" />
            </button>
            <button className="wh-modal-cancel" onClick={() => setPage(totalPages)} disabled={page === totalPages} title="หน้าสุดท้าย" aria-label="หน้าสุดท้าย">
              <ChevronDoubleRightIcon className="size-4" />
            </button>
          </div>
        </div>}
    </>;
}
