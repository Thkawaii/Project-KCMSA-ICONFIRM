import { useEffect, useMemo, useRef, useState } from 'react';
import { getPartChecks, scanPartCheck, deletePartCheck } from '../api/partcheck.js';
import { getImportLicenseItems } from '../api/importLicense.js';
import { API_BASE_URL } from '../api/client.js';
import { scanStep, scanSelect, scanLoading, scanSuccessToast, scanErrorAlert, scanClose } from '../lib/scanPopup.js';
import { confirmDelete, toastSuccess, toastError } from '../lib/toast.js';
import { inDateTab } from '../lib/dateRange.js';
import { CheckIcon, ChevronDoubleLeftIcon, ChevronDoubleRightIcon, ChevronLeftIcon, ChevronRightIcon, ClockIcon, DocumentTextIcon, ExclamationTriangleIcon, MinusIcon, PART_ICONS_BY_CODE, ShieldCheckIcon, TagIcon, XMarkIcon } from '../components/icons.jsx';
import AppShell from '../components/AppShell.jsx';
import SelectField from '../components/Selectfield.jsx';
import PartTag from '../components/Parttag.jsx';
import { WH_NAV_ITEMS } from './Importlicensepage.jsx';
import bcItc from '../assets/barcodes/IT_Controller.gif';
import bcSwingSn from '../assets/barcodes/Swing_Motor__SN_.gif';
import bcPumpSn from '../assets/barcodes/Pump_Assy_HYD__SN_.gif';
import bcMotorSn from '../assets/barcodes/Motor_Propel__SN_.gif';
import bcValveSn from '../assets/barcodes/Control_Valve__SN_.gif';
import bcEngine from '../assets/barcodes/Engine.gif';
import bcCounterWeight from '../assets/barcodes/Counter_Weight.gif';
const TAG_TYPES = [{
  code: 'MC',
  label: 'Machine',
  needsPN: false
}, {
  code: 'ITC',
  label: 'IT Controller',
  needsPN: true
}, {
  code: 'CV',
  label: 'Control Valve',
  needsPN: false
}, {
  code: 'SM',
  label: 'Swing Motor',
  needsPN: false
}, {
  code: 'MP',
  label: 'Motor Propel',
  needsPN: false
}, {
  code: 'PH',
  label: 'Pump Assy HYD',
  needsPN: false
}, {
  code: 'EN',
  label: 'Engine',
  needsPN: true
}, {
  code: 'CW',
  label: 'Counter Weight',
  needsPN: false,
  snLabel: 'CounterWeight No.'
}];
const PART_TYPES = TAG_TYPES.filter(t => t.code !== 'MC');
function tagLabel(code) {
  return TAG_TYPES.find(t => t.code === code)?.label || code || '—';
}
function firstToken(v) {
  if (!v) return '';
  return String(v).trim().split(/\s+/)[0] || '';
}
const MATCH_LABELS = {
  MATCH: {
    Icon: CheckIcon,
    text: 'ตรงกับใบอนุญาต',
    cls: 'il-badge-ok'
  },
  NOT_FOUND: {
    Icon: XMarkIcon,
    text: 'ไม่พบในใบอนุญาต',
    cls: 'il-badge-bad'
  },
  WRONG_INVOICE: {
    Icon: ExclamationTriangleIcon,
    text: 'คนละอินวอยซ์',
    cls: 'il-badge-warn'
  },
  WRONG_PRODNO: {
    Icon: ExclamationTriangleIcon,
    text: 'หมายเลขการผลิตไม่ตรง',
    cls: 'il-badge-warn'
  },
  WRONG_PART: {
    Icon: XMarkIcon,
    text: 'ข้อมูลไม่ตรง',
    cls: 'il-badge-bad'
  },
  DUPLICATE: {
    Icon: ExclamationTriangleIcon,
    text: 'ยืนยันซ้ำ',
    cls: 'il-badge-warn'
  },
  NOT_REQUIRED: {
    Icon: MinusIcon,
    text: 'ไม่ต้องเทียบ',
    cls: 'il-badge-muted'
  }
};
const NON_LICENSE_MATCH_TEXT = {
  MATCH: 'ข้อมูลถูกต้อง',
  NOT_FOUND: 'ข้อมูลไม่ถูกต้อง',
  WRONG_PART: 'ข้อมูลไม่ถูกต้อง',
  WRONG_INVOICE: 'ข้อมูลไม่ถูกต้อง',
  WRONG_PRODNO: 'ข้อมูลไม่ถูกต้อง'
};
function matchBadge(status, partType) {
  const m = MATCH_LABELS[status] || MATCH_LABELS.NOT_REQUIRED;
  const code = String(partType || '').toUpperCase();
  const text = code && code !== 'ITC' && NON_LICENSE_MATCH_TEXT[status] ? NON_LICENSE_MATCH_TEXT[status] : m.text;
  return <span className={'il-badge ' + m.cls}>
      <m.Icon className="inline size-3.5 align-text-bottom" /> {text}
    </span>;
}
const BARCODE_CARDS = [{
  partType: 'ITC',
  title: 'IT Controller',
  caption: 'IT Controller',
  img: bcItc,
  kind: 'P/N + S/N'
}, {
  partType: 'SM',
  title: 'Swing Motor',
  caption: 'Swing Motor (S/N)',
  img: bcSwingSn,
  kind: 'S/N'
}, {
  partType: 'PH',
  title: 'Pump Assy HYD',
  caption: 'Pump Assy HYD (S/N)',
  img: bcPumpSn,
  kind: 'S/N'
}, {
  partType: 'MP',
  title: 'Motor Propel',
  caption: 'Motor Propel (S/N)',
  img: bcMotorSn,
  kind: 'S/N'
}, {
  partType: 'CV',
  title: 'Control Valve',
  caption: 'Control Valve (S/N)',
  img: bcValveSn,
  kind: 'S/N'
}, {
  partType: 'EN',
  title: 'Engine',
  caption: 'Engine',
  img: bcEngine,
  kind: 'P/N + S/N'
}, {
  partType: 'CW',
  title: 'Counter Weight',
  caption: 'Counter Weight',
  img: bcCounterWeight,
  kind: 'CounterWeight No.'
}];
export default function WHPartConfirmationPage() {
  const isManager = (localStorage.getItem('iconfirm_role') || '').toUpperCase() === 'LOG';
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [licenseItems, setLicenseItems] = useState([]);
  const [licenseTab, setLicenseTab] = useState('all');
  const [licenseModel, setLicenseModel] = useState('all');
  const [licenseNo, setLicenseNo] = useState('all');
  const [licensePageSize, setLicensePageSize] = useState(10);
  const [licensePage, setLicensePage] = useState(1);
  const [highlightId, setHighlightId] = useState(null);
  const [dateTab, setDateTab] = useState('all');
  const [search, setSearch] = useState('');
  const [matchFilter, setMatchFilter] = useState('all');
  const [pageSize, setPageSize] = useState(10);
  const [page, setPage] = useState(1);
  const [detailRow, setDetailRow] = useState(null);
  const busyRef = useRef(false);
  const fireRef = useRef(() => {});
  const armedPartRef = useRef(null);
  const [armedPart, setArmedPart] = useState(null);
  async function loadRows() {
    setLoading(true);
    setLoadError('');
    try {
      const [checks, items] = await Promise.all([getPartChecks(), isManager ? getImportLicenseItems() : Promise.resolve([])]);
      setRows(checks || []);
      setLicenseItems(items || []);
    } catch (err) {
      setLoadError(err.message || 'โหลดข้อมูลไม่สำเร็จ');
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    loadRows();
  }, []);
  useEffect(() => {
    setPage(1);
  }, [dateTab, search, matchFilter, pageSize]);
  useEffect(() => {
    setLicensePage(1);
  }, [licenseTab, licenseModel, licenseNo, licensePageSize]);
  async function handleDeleteCheck(row) {
    const label = `${tagLabel(row.PartType)} — ${row.SN || row.PN || '#' + row.ID}`;
    const ok = await confirmDelete({
      text: `ลบรายการสแกน ${label} ออกจากประวัติ? กู้คืนไม่ได้`
    });
    if (!ok) return;
    try {
      await deletePartCheck(row.ID);
      toastSuccess(`ลบรายการ ${label} แล้ว`);
      await loadRows();
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    }
  }
  async function runScanFlow(partTypeCode) {
    if (!partTypeCode || busyRef.current) return;
    const part = PART_TYPES.find(t => t.code === partTypeCode);
    if (!part) return;
    const partLabel = part.label;
    const isITC = part.code === 'ITC';
    const needsPN = Boolean(part.needsPN);
    armedPartRef.current = partTypeCode;
    setArmedPart(partTypeCode);
    busyRef.current = true;
    let successToast = null;
    try {
      let pn = '';
      if (needsPN) {
        pn = firstToken(await scanStep({
          title: `${partLabel}( P/N)`,
          placeholder: 'ยิงบาร์โค้ด หรือพิมพ์ P/N แล้วกดปุ่ม',
          html: ''
        }));
        if (!pn) return;
      }
      const snLabel = part.snLabel || 'S/N';
      const sn = firstToken(await scanStep({
        title: `${partLabel}( ${snLabel})`,
        placeholder: `ยิงบาร์โค้ด หรือพิมพ์ ${snLabel} แล้วกดปุ่ม`,
        html: needsPN ? `<div class="scan-popup-hint">P/N: <b>${pn}</b></div>` : '',
        confirmText: 'บันทึก'
      }));
      if (!sn) return;
      if (needsPN && sn === pn) {
        await scanErrorAlert(`ค่า S/N ซ้ำกับ P/N (${sn}) — เหมือนสแกนบาร์โค้ดเดิมซ้ำ กรุณาสแกน S/N ของ ${partLabel} อีกครั้ง`);
        return;
      }
      scanLoading('กำลังตรวจสอบกับบัญชีใบอนุญาต...');
      try {
        const res = await scanPartCheck({
          partType: partTypeCode,
          pn: needsPN ? pn : '',
          sn,
          productionNo: '',
          invoiceNo: ''
        });
        const check = res.check || res;
        if (res.item?.ID) {
          setHighlightId(res.item.ID);
          setTimeout(() => setHighlightId(null), 6000);
        }
        if (res.matched) {
          successToast = isITC ? `ตรงกับบัญชี: ${sn}` : `บันทึกแล้ว: ${tagLabel(check.PartType)} — ${sn}`;
        } else if (isITC) {
          const errMsg = check.MatchMessage || res.message || 'ไม่ตรงกับบัญชีใบอนุญาตนำเข้า';
          const isMasterDataMiss = check.MatchStatus === 'NOT_FOUND' && errMsg.includes('ทะเบียนกลาง');
          await scanErrorAlert(errMsg, isMasterDataMiss ? {
            hint: 'กรุณาติดต่อ ADMIN'
          } : undefined);
        } else if (check.MatchStatus === 'NOT_REQUIRED') {
          successToast = `บันทึกแล้ว: ${tagLabel(check.PartType)} — ${sn}`;
        } else {
          await scanErrorAlert('ข้อมูลไม่ถูกต้อง');
        }
        await loadRows();
      } catch (err) {
        await scanErrorAlert(err.message || 'บันทึกไม่สำเร็จ');
      }
    } finally {
      busyRef.current = false;
      scanClose();
    }
    if (successToast) scanSuccessToast(successToast);
  }
  function detectPartType(raw) {
    const s = (raw || '').toUpperCase();
    if (s.includes('SWING')) return 'SM';
    if (s.includes('PROPEL')) return 'MP';
    if (s.includes('PUMP') || s.includes('HYD')) return 'PH';
    if (s.includes('CONTROL VALVE') || s.includes('VALVE')) return 'CV';
    if (s.includes('CONTROLLER')) return 'ITC';
    if (s.includes('ENGINE')) return 'EN';
    if (s.includes('COUNTER') || s.includes('COUNTERWEIGHT')) return 'CW';
    return null;
  }
  async function handleScannerFire(code) {
    if (busyRef.current) return;
    let partType = detectPartType(code);
    if (!partType && armedPartRef.current) {
      partType = armedPartRef.current;
    }
    if (!partType) {
      busyRef.current = true;
      let picked = null;
      try {
        picked = await scanSelect({
          title: 'เลือกชนิดพาร์ทที่จะยืนยัน',
          html: `<div class="scan-popup-hint">บาร์โค้ดที่ยิง: <b>${code}</b></div>`,
          options: PART_TYPES.map(p => ({
            value: p.code,
            label: p.label
          }))
        });
      } finally {
        busyRef.current = false;
      }
      if (!picked) return;
      partType = picked;
    }
    runScanFlow(partType);
  }
  fireRef.current = handleScannerFire;
  useEffect(() => {
    let buffer = '';
    let lastTime = 0;
    let flushTimer = null;
    let startedClean = false;
    function fireBuffered() {
      if (flushTimer) {
        clearTimeout(flushTimer);
        flushTimer = null;
      }
      const code = buffer.trim();
      const clean = startedClean;
      buffer = '';
      startedClean = false;
      if (busyRef.current) return;
      if (clean && code.length >= 2) fireRef.current(code);
    }
    function onKeydown(e) {
      if (busyRef.current) {
        lastTime = Date.now();
        buffer = '';
        startedClean = false;
        return;
      }
      const now = Date.now();
      const gap = now - lastTime;
      lastTime = now;
      if (gap > 50) {
        buffer = '';
        startedClean = true;
      }
      if (e.key === 'Enter') {
        if (startedClean && buffer.trim().length >= 2) {
          e.preventDefault();
          fireBuffered();
        } else {
          buffer = '';
          startedClean = false;
        }
        return;
      }
      if (e.key && e.key.length === 1) {
        buffer += e.key;
        if (buffer.length >= 2) e.preventDefault();
        if (flushTimer) clearTimeout(flushTimer);
        flushTimer = setTimeout(fireBuffered, 120);
      }
    }
    function onGlobalInput(e) {
      if (busyRef.current) return;
      const inserted = typeof e.data === 'string' ? e.data : '';
      const code = inserted.trim();
      if (code.length < 2) return;
      const target = e.target;
      if (target && typeof target.value === 'string') {
        try {
          target.value = target.value.slice(0, Math.max(0, target.value.length - inserted.length));
        } catch {}
      }
      buffer = '';
      startedClean = false;
      fireRef.current(code);
    }
    window.addEventListener('keydown', onKeydown);
    window.addEventListener('input', onGlobalInput, true);
    return () => {
      window.removeEventListener('keydown', onKeydown);
      window.removeEventListener('input', onGlobalInput, true);
      if (flushTimer) clearTimeout(flushTimer);
    };
  }, []);
  const licenseRows = useMemo(() => {
    let list = licenseItems;
    if (licenseTab === 'pending') list = list.filter(r => r.ConfirmStatus !== 'CONFIRMED');
    if (licenseTab === 'confirmed') list = list.filter(r => r.ConfirmStatus === 'CONFIRMED');
    if (licenseModel !== 'all') list = list.filter(r => (r.Model || '') === licenseModel);
    if (licenseNo !== 'all') list = list.filter(r => (r.LicenseNo || '') === licenseNo);
    return list;
  }, [licenseItems, licenseTab, licenseModel, licenseNo]);
  const licenseTotalPages = Math.max(1, Math.ceil(licenseRows.length / licensePageSize));
  const licensePaged = licenseRows.slice((licensePage - 1) * licensePageSize, licensePage * licensePageSize);
  function goToLicensePage(p) {
    setLicensePage(Math.min(Math.max(1, p), licenseTotalPages));
  }
  useEffect(() => {
    if (!highlightId) return;
    const idx = licenseRows.findIndex(r => r.ID === highlightId);
    if (idx >= 0) setLicensePage(Math.floor(idx / licensePageSize) + 1);
  }, [highlightId, licenseRows, licensePageSize]);
  const licenseModelOptions = useMemo(() => {
    const set = new Set();
    licenseItems.forEach(r => {
      if (r.Model) set.add(r.Model);
    });
    return Array.from(set).sort();
  }, [licenseItems]);
  const licenseNoOptions = useMemo(() => {
    const set = new Set();
    licenseItems.forEach(r => {
      if (r.LicenseNo) set.add(r.LicenseNo);
    });
    return Array.from(set).sort();
  }, [licenseItems]);
  const licenseCounts = useMemo(() => {
    return {
      total: licenseItems.length,
      confirmed: licenseItems.filter(r => r.ConfirmStatus === 'CONFIRMED').length,
      pending: licenseItems.filter(r => r.ConfirmStatus !== 'CONFIRMED').length
    };
  }, [licenseItems]);
  const filtered = useMemo(() => {
    let list = rows;
    if (dateTab !== 'all') {
      list = list.filter(r => inDateTab(r.CheckedDatetime, dateTab));
    }
    if (matchFilter !== 'all') {
      list = list.filter(r => r.MatchStatus === matchFilter);
    }
    const term = search.trim().toLowerCase();
    if (term) {
      list = list.filter(r => (r.PN || '').toLowerCase().includes(term) || (r.SN || '').toLowerCase().includes(term) || (r.MachineNo || '').toLowerCase().includes(term) || (r.CheckedBy || '').toLowerCase().includes(term));
    }
    return list;
  }, [rows, dateTab, search, matchFilter]);
  const mismatchCount = useMemo(() => rows.filter(r => r.PartType === 'ITC' && r.MatchStatus && r.MatchStatus !== 'MATCH').length, [rows]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages));
  }
  return <AppShell navItems={WH_NAV_ITEMS} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">{isManager ? 'Part Checklist' : 'Part Confirmation'}</h2>
        </div>
      </div>

      {loadError && <p className="form-error" role="alert">
          {loadError}
        </p>}

      {!isManager && <>
          <div className="pc-barcode-grid">
            {BARCODE_CARDS.map(card => <div className={'pc-barcode-card pc-card-' + card.partType.toLowerCase() + (armedPart === card.partType ? ' pc-barcode-card-armed' : '')} key={card.partType} role="button" tabIndex={0} title={`เริ่มสแกน ${card.title}`} onClick={() => runScanFlow(card.partType)} onKeyDown={e => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            runScanFlow(card.partType);
          }
        }}>
                {armedPart === card.partType && <span className="pc-barcode-armed-tag">โหมดสแกน</span>}
                <span className="pc-barcode-kind">{card.kind}</span>
                <div className="pc-barcode-title">{card.title}</div>
                <div className="pc-barcode-box">
                  <img className="pc-barcode-img" src={card.img} alt={`บาร์โค้ด ${card.caption}`} />
                </div>
              </div>)}
          </div>

        </>}

      {isManager && <>
          {!loading && licenseItems.length === 0 && <p className="wh-subtitle">
              ยังไม่มีบัญชีใบอนุญาตนำเข้าในระบบ — ไปที่เมนู <strong>Import License</strong>{' '}
              เพื่ออัปโหลดไฟล์ Excel ก่อน แล้วค่อยกลับมาสแกน
            </p>}
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{
            fontSize: 19
          }}>
            เทียบกับบัญชีใบอนุญาตนำเข้า ({licenseCounts.confirmed}/{licenseCounts.total})
          </h2>
        </div>
        <div className="vr-tabs">
          {[{
            key: 'all',
            label: `ทั้งหมด (${licenseCounts.total})`
          }, {
            key: 'pending',
            label: `รอสแกน (${licenseCounts.pending})`
          }, {
            key: 'confirmed',
            label: `ยืนยันแล้ว (${licenseCounts.confirmed})`
          }].map(tab => <button key={tab.key} className={'vr-tab' + (licenseTab === tab.key ? ' vr-tab-active' : '')} onClick={() => setLicenseTab(tab.key)}>
              {tab.label}
            </button>)}
        </div>
      </div>

      <div className="tsf-history-toolbar">
        <div className="wh-history-filters">
          <div className="tsf-history-pagesize">
            <div className="wh-pagesize-select">
              <SelectField value={licensePageSize} onChange={setLicensePageSize} options={[{
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
          <div className="wh-filter-field">
            <span className="wh-filter-label">แบบ/รุ่น</span>
            <SelectField value={licenseModel} onChange={setLicenseModel} options={[{
              value: 'all',
              label: 'ทั้งหมด'
            }, ...licenseModelOptions.map(m => ({
              value: m,
              label: m
            }))]} />
          </div>
          <div className="wh-filter-field">
            <span className="wh-filter-label">ใบอนุญาตนำเข้า</span>
            <SelectField value={licenseNo} onChange={setLicenseNo} options={[{
              value: 'all',
              label: 'ทั้งหมด'
            }, ...licenseNoOptions.map(n => ({
              value: n,
              label: n
            }))]} />
          </div>
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ลำดับ</th>
              <th>แบบ/รุ่น</th>
              <th>ใบอนุญาตนำเข้า</th>
              <th>อินวอยซ์</th>
              <th>หมายเลขเครื่อง</th>
              <th>หมายเลขการผลิต</th>
              <th>หมายเหตุ</th>
              <th>ส่งออกไปประเทศ</th>
              <th>สถานะ</th>
              <th>ยืนยันเมื่อ</th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={10} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>}
            {!loading && licensePaged.map((r, idx) => <tr key={r.ID} className={highlightId === r.ID ? 'il-row-hit' : ''}>
                  <td className="wh-cell-head" data-label="ลำดับ">
                    {(licensePage - 1) * licensePageSize + idx + 1}
                  </td>
                  <td data-label="แบบ/รุ่น">{r.Model || '—'}</td>
                  <td data-label="ใบอนุญาตนำเข้า">{r.LicenseNo || '—'}</td>
                  <td data-label="อินวอยซ์">{r.InvoiceNo || '—'}</td>
                  <td className="il-mono" data-label="หมายเลขเครื่อง">
                    <strong>{r.MachineNo}</strong>
                  </td>
                  <td className="il-mono" data-label="หมายเลขการผลิต">
                    {r.ProductionNo || '—'}
                  </td>
                  <td data-label="หมายเหตุ">{r.Remark || '—'}</td>
                  <td data-label="ส่งออกไปประเทศ">{r.ExportCountry || '—'}</td>
                  <td data-label="สถานะ">
                    {r.ConfirmStatus === 'CONFIRMED' ? <span className="il-badge il-badge-ok">
                        <CheckIcon className="inline size-3.5 align-text-bottom" /> ตรงกัน
                      </span> : <span className="il-badge il-badge-pending">
                        <ClockIcon className="inline size-3.5 align-text-bottom" /> รอสแกน
                      </span>}
                  </td>
                  <td data-label="ยืนยันเมื่อ">
                    {r.ConfirmedDatetime ? new Date(r.ConfirmedDatetime).toLocaleString('th-TH') : '—'}
                  </td>
                </tr>)}
            {!loading && licenseRows.length === 0 && <tr>
                <td colSpan={10} className="wh-empty-cell">
                  ไม่มีรายการในมุมมองนี้
                </td>
              </tr>}
          </tbody>
        </table>
      </div>

      {!loading && licenseRows.length > 0 && <div className="tsf-pagination">
          <span className="wh-subtitle" style={{
          fontSize: 13
        }}>
            Showing {(licensePage - 1) * licensePageSize + 1} to{' '}
            {Math.min(licensePage * licensePageSize, licenseRows.length)} of {licenseRows.length}{' '}
            entries
          </span>
          <div className="tsf-pagination-buttons">
            <button className="wh-modal-cancel" onClick={() => goToLicensePage(1)} disabled={licensePage === 1}>
              <ChevronDoubleLeftIcon className="size-4" />
            </button>
            <button className="wh-modal-cancel" onClick={() => goToLicensePage(licensePage - 1)} disabled={licensePage === 1}>
              <ChevronLeftIcon className="size-4" />
            </button>
            <span className="tsf-pagination-current">
              {licensePage} / {licenseTotalPages}
            </span>
            <button className="wh-modal-cancel" onClick={() => goToLicensePage(licensePage + 1)} disabled={licensePage === licenseTotalPages}>
              <ChevronRightIcon className="size-4" />
            </button>
            <button className="wh-modal-cancel" onClick={() => goToLicensePage(licenseTotalPages)} disabled={licensePage === licenseTotalPages}>
              <ChevronDoubleRightIcon className="size-4" />
            </button>
          </div>
        </div>}
        </>}

      {!isManager && <>
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{
            fontSize: 19
          }}>
            ประวัติการสแกน ({filtered.length})
          </h2>
          {mismatchCount > 0 && <p className="wh-subtitle" style={{
            color: '#b42318',
            fontWeight: 600
          }}>
              มี {mismatchCount} รายการที่สแกนแล้วไม่ตรงกับบัญชีใบอนุญาต
            </p>}
        </div>
        <div className="vr-tabs">
          {[{
            key: 'all',
            label: 'ทั้งหมด'
          }, {
            key: 'day',
            label: 'รายวัน'
          }, {
            key: 'week',
            label: 'รายสัปดาห์'
          }, {
            key: 'month',
            label: 'รายเดือน'
          }].map(tab => <button key={tab.key} className={'vr-tab' + (dateTab === tab.key ? ' vr-tab-active' : '')} onClick={() => setDateTab(tab.key)}>
              {tab.label}
            </button>)}
        </div>
      </div>

      <div className="tsf-history-toolbar">
        <div className="wh-history-filters">
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
          <div className="wh-filter-field">
            <span className="wh-filter-label">ผลการตรวจสอบ</span>
            <SelectField value={matchFilter} onChange={setMatchFilter} options={[{
              value: 'all',
              label: 'ทั้งหมด'
            }, {
              value: 'MATCH',
              label: 'ตรงกับใบอนุญาต'
            }, {
              value: 'NOT_FOUND',
              label: 'ไม่พบในใบอนุญาต'
            }, {
              value: 'NOT_REQUIRED',
              label: 'ไม่ต้องเทียบ'
            }, {
              value: 'DUPLICATE',
              label: 'ยืนยันซ้ำ'
            }]} />
          </div>
        </div>
        <input className="wh-search" type="text" placeholder="ค้นหา Tag / P/N / หมายเลขเครื่อง / ผู้ตรวจสอบ" value={search} onChange={e => setSearch(e.target.value)} />
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ITEM</th>
              <th>Part</th>
              <th>P/N</th>
              <th>S/N</th>
              <th>หมายเลขเครื่อง</th>
              <th>ผลการตรวจสอบ</th>
              <th>Checked By</th>
              <th>วันที่</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={10} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>}
            {!loading && paged.map((r, idx) => <tr key={r.ID}>
                  <td className="wh-cell-head" data-label="ITEM">
                    {(page - 1) * pageSize + idx + 1}
                  </td>
                  <td className="wh-cell-head" data-label="Part">
                    {tagLabel(r.PartType)}
                  </td>
                  <td className="il-mono" data-label="P/N">
                    {r.PN || '—'}
                    {r.MatchStatus === 'WRONG_PART' && r.ExpectedPN && <span className="mfg-plan-hint" title={r.MatchDetail || ''}>
                        แผน: {r.ExpectedPN}
                      </span>}
                  </td>
                  <td className="il-mono" data-label="S/N">
                    {r.SN || '—'}
                  </td>
                  <td className="il-mono" data-label="หมายเลขเครื่อง">
                    {r.MachineNo || '—'}
                  </td>
                  <td data-label="ผลการตรวจสอบ">{matchBadge(r.MatchStatus, r.PartType)}</td>
                  <td data-label="Checked By">{r.CheckedBy}</td>
                  <td data-label="วันที่">{new Date(r.CheckedDatetime).toLocaleString('th-TH')}</td>
                  <td className="wh-cell-action">
                    <button className="tsf-action-btn" onClick={() => setDetailRow(r)}>
                      รายละเอียด
                    </button>
                    {['NOT_FOUND', 'NOT_REQUIRED', 'DUPLICATE', 'WRONG_PART', 'WRONG_INVOICE', 'WRONG_PRODNO'].includes(r.MatchStatus) && <button className="tsf-action-btn tsf-action-btn-danger" onClick={() => handleDeleteCheck(r)}>
                        ลบ
                      </button>}
                  </td>
                </tr>)}
            {!loading && paged.length === 0 && <tr>
                <td colSpan={10} className="wh-empty-cell">
                  ยังไม่มีรายการตรวจสอบ
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
        </>}

      {detailRow && <div className="wh-modal-overlay" onClick={() => setDetailRow(null)}>
          <div className="wh-modal wh-detail-modal" onClick={e => e.stopPropagation()}>
            <button type="button" className="wh-detail-close" onClick={() => setDetailRow(null)} aria-label="ปิด">
              <XMarkIcon className="size-4" />
            </button>

            <div className="wh-detail-header">
              <span className="wh-detail-header-icon">
                {(() => {
              const PartIcon = PART_ICONS_BY_CODE[detailRow.PartType] || TagIcon;
              return <PartIcon className="size-5" />;
            })()}
              </span>
              <div>
                <h3 className="wh-modal-title">รายละเอียดการตรวจสอบ</h3>
                <span className="wh-detail-header-sub">{tagLabel(detailRow.PartType)}</span>
              </div>
            </div>

            <div className="wh-detail-section">
              <span className="wh-detail-section-title">
                <DocumentTextIcon className="size-4" /> ข้อมูลชิ้นงาน
              </span>
              <div className="wh-detail-grid">
                <div className="wh-detail-item">
                  <span className="wh-detail-label">P/N</span>
                  <span className="wh-detail-value mono">{detailRow.PN || '—'}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">S/N</span>
                  <span className="wh-detail-value mono">{detailRow.SN || '—'}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">หมายเลขเครื่อง</span>
                  <span className="wh-detail-value mono wh-detail-value-tagged">
                    <span>{detailRow.MachineNo || '—'}</span>
                    <PartTag code={detailRow.PartType} />
                  </span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">หมายเลขการผลิต (IMEI)</span>
                  <span className="wh-detail-value mono">{detailRow.ProductionNo || '—'}</span>
                </div>
              </div>
            </div>

            <div className="wh-detail-divider" />

            <div className="wh-detail-section">
              <span className="wh-detail-section-title">
                <ShieldCheckIcon className="size-4" /> ผลตรวจสอบใบอนุญาต
              </span>
              <div className="wh-detail-grid">
                <div className="wh-detail-item">
                  <span className="wh-detail-label">ใบอนุญาตนำเข้า</span>
                  <span className="wh-detail-value mono">{detailRow.LicenseNo || '—'}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">อินวอยซ์</span>
                  <span className="wh-detail-value mono">{detailRow.InvoiceNo || '—'}</span>
                </div>
              </div>
              <div className="wh-detail-result">
                {matchBadge(detailRow.MatchStatus, detailRow.PartType)}
                {detailRow.MatchMessage ? <span className="wh-detail-result-msg">{detailRow.MatchMessage}</span> : null}
              </div>
              {detailRow.MatchDetail ? <div className="wh-detail-result-detail">{detailRow.MatchDetail}</div> : null}
            </div>

            <div className="wh-detail-meta">
              <span>
                <TagIcon className="size-3.5" /> ตรวจสอบโดย {detailRow.CheckedBy}
              </span>
              <span>
                <ClockIcon className="size-3.5" />{' '}
                {new Date(detailRow.CheckedDatetime).toLocaleString('th-TH')}
              </span>
            </div>

            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={() => setDetailRow(null)}>
                ปิด
              </button>
            </div>
          </div>
        </div>}
    </AppShell>;
}
