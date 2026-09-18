import { useEffect, useMemo, useRef, useState } from 'react';
import { getMFGAssemblies, scanMFGAssembly, createMFGAssembly, updateMFGAssembly, deleteMFGAssembly, uploadMFGAssemblyPhoto, checkMFGSpecQR, looksLikeSpecQR, specQRMachineNo } from '../api/mfgAssembly.js';
import { API_BASE_URL } from '../api/client.js';
import { getUploadData } from '../api/uploadData.js';
import { getMachinePlans, indexMachinePlans, lookupMachinePlan } from '../api/machinePlans.js';
import { confirmDelete, toastSuccess, toastError } from '../lib/toast.js';
import { inPeriod } from '../lib/dateRange.js';
import PeriodRangePicker from '../components/PeriodRangePicker.jsx';
import { scanStep, scanLoading, scanClose, scanCloseWait, scanSuccessToast, scanErrorAlert, scanPhotoCapture, scanSpecAlert } from '../lib/scanPopup.js';
import { ChevronDoubleLeftIcon, ChevronDoubleRightIcon, ChevronLeftIcon, ChevronRightIcon, QrCodeIcon, CameraIcon, ArrowUpTrayIcon, ArrowsRightLeftIcon, XMarkIcon, DocumentTextIcon, CubeIcon, ClockIcon, TagIcon, WrenchScrewdriverIcon } from '../components/icons.jsx';
import AppShell from '../components/AppShell.jsx';
import SelectField from '../components/Selectfield.jsx';
import bcKanban from '../assets/barcodes/Kanban.gif';
function escapeHtml(v) {
  return String(v ?? '').replace(/[&<>"']/g, ch => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  })[ch]);
}
function firstToken(v) {
  if (!v) return '';
  return String(v).trim().split(/\s+/)[0] || '';
}
function normCode(v) {
  return String(v || '').toUpperCase().replace(/[^A-Z0-9]/g, '');
}
function sameCode(a, b) {
  const na = normCode(a);
  return na !== '' && na === normCode(b);
}
export const MFG_NAV_ITEMS = [{
  to: '/mfg-assembly',
  label: 'MFG Assembly',
  icon: <ArrowsRightLeftIcon className="size-4" />,
  roles: ['MFG']
}];
const STATUS_META = {
  MATCHED: {
    label: 'MATCHED',
    cls: 'il-badge il-badge-ok'
  },
  NOT_MATCHED: {
    label: 'NOT_MATCHED',
    cls: 'il-badge il-badge-bad'
  },
  DUPLICATE: {
    label: 'DUPLICATE',
    cls: 'il-badge il-badge-warn'
  },
  RETIRED_FORMAT: {
    label: 'รูปแบบเดิมถูกยกเลิก',
    cls: 'il-badge il-badge-bad'
  },
  OK: {
    label: 'ตรงกัน',
    cls: 'il-badge il-badge-ok'
  },
  UNKNOWN: {
    label: 'ไม่พบในทะเบียน',
    cls: 'il-badge il-badge-warn'
  },
  REUSED: {
    label: 'ผูกกับเครื่องอื่น',
    cls: 'il-badge il-badge-bad'
  }
};
const STATUS_OPTIONS = [{
  value: 'NOT_MATCHED',
  label: 'NOT_MATCHED — ยังไม่ตรง/ยังไม่ยืนยัน'
}, {
  value: 'DUPLICATE',
  label: 'DUPLICATE — ซ้ำ'
}];
const EMPTY_FORM = {
  dateAssembly: '',
  machineNo: '',
  itControllerNo: '',
  partNo: '',
  serialNo: '',
  country: '',
  checkDate: '',
  status: ''
};
function fmtDate(value) {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  const buddhistYear = d.getFullYear() + 543;
  const day = d.getDate();
  const month = d.getMonth() + 1;
  const pad = n => String(n).padStart(2, '0');
  const time = `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
  return `${day}/${month}/${buddhistYear} ${time}`;
}
function toDateInput(value) {
  if (!value) return '';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '';
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${m}-${day}`;
}
function parseAssemblyCode(raw) {
  const s = (raw || '').trim();
  if (!s) return {
    machineNo: '',
    itControllerNo: ''
  };
  const tokens = s.split(/[\s,;|]+/).map(t => t.trim()).filter(Boolean);
  const itc = tokens.find(t => /^\d{10,15}$/.test(t)) || '';
  const mc = tokens.find(t => t !== itc) || '';
  if (itc || mc) return {
    machineNo: mc,
    itControllerNo: itc
  };
  return {
    machineNo: tokens[0] || '',
    itControllerNo: tokens[1] || ''
  };
}
// ผลเทียบ QR กับ Specification sheet ที่บันทึกไว้ในแถว (SpecDetail = JSON)
function parseSpecDetail(row) {
  if (!row?.SpecDetail) return null;
  try {
    return JSON.parse(row.SpecDetail);
  } catch {
    return null;
  }
}
// IT device ของเครื่อง: จาก Daily Plan ก่อน ถ้าไม่มีใช้ค่าจาก Kanban (QR) ที่สแกน
function itDeviceOf(row, asm) {
  if (asm?.itDevice) return asm.itDevice;
  const spec = parseSpecDetail(row);
  const item = (spec?.items || []).find(it => it.field === 'itDevice');
  return item ? item.plan || item.qr || '' : '';
}
function todayYMD() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
export default function MFGAssemblyPage() {
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [search, setSearch] = useState('');
  const [periodMode, setPeriodMode] = useState('all');
  const [periodAnchor, setPeriodAnchor] = useState('');
  function handlePeriodModeChange(next) {
    setPeriodMode(next);
    if (next !== 'all' && !periodAnchor) setPeriodAnchor(todayYMD());
  }
  function clearPeriod() {
    setPeriodMode('all');
    setPeriodAnchor('');
  }
  const [pageSize, setPageSize] = useState(10);
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editId, setEditId] = useState(null);
  const [form, setForm] = useState(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [photoView, setPhotoView] = useState(null);
  const [photoBusy, setPhotoBusy] = useState(false);
  const [photoEditRow, setPhotoEditRow] = useState(null);
  const photoFileInputRef = useRef(null);
  const pendingPhotoRowIdRef = useRef(null);
  const [planIndex, setPlanIndex] = useState(null);
  const [detailRow, setDetailRow] = useState(null);
  // ข้อมูล Spec sheet (Daily Plan) ของเครื่องที่เปิดดูรายละเอียด
  const [detailPlan, setDetailPlan] = useState({
    loading: false,
    columns: [],
    data: null
  });
  useEffect(() => {
    const mc = detailRow?.row?.MachineNo;
    if (!mc) return undefined;
    let cancelled = false;
    setDetailPlan({
      loading: true,
      columns: [],
      data: null
    });
    getUploadData('daily_plan', mc, 1, 20).then(res => {
      if (cancelled) return;
      const want = normCode(mc);
      const hit = (res?.rows || []).find(r => normCode(r.MachineNo) === want);
      let data = null;
      if (hit) {
        try {
          data = JSON.parse(hit.DataJSON || '{}');
        } catch {
          data = null;
        }
      }
      setDetailPlan({
        loading: false,
        columns: res?.columns || [],
        data
      });
    }).catch(() => {
      if (!cancelled) setDetailPlan({
        loading: false,
        columns: [],
        data: null
      });
    });
    return () => {
      cancelled = true;
    };
  }, [detailRow]);
  const [scanBusy, setScanBusy] = useState(false);
  const busyRef = useRef(false);
  const fireRef = useRef(() => {});
  function friendlyError(err, fallback) {
    if (err?.status === 404 || err?.status === 405) {
      return 'ยังไม่พบ API /mfg-assembly ที่ฝั่งเซิร์ฟเวอร์ — ต้อง rebuild แล้ว restart backend ก่อน';
    }
    return err?.message || fallback;
  }
  async function loadRows() {
    setLoading(true);
    setLoadError('');
    try {
      const list = await getMFGAssemblies();
      setRows(list || []);
    } catch (err) {
      setLoadError(friendlyError(err, 'โหลดข้อมูลไม่สำเร็จ'));
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    loadRows();
  }, []);
  useEffect(() => {
    let cancelled = false;
    async function loadMachinePlans() {
      try {
        const data = await getMachinePlans();
        if (!cancelled) setPlanIndex(indexMachinePlans(data?.rows || []));
      } catch {
        if (!cancelled) setPlanIndex(null);
      }
    }
    loadMachinePlans();
    return () => {
      cancelled = true;
    };
  }, []);
  function assemblyFor(row) {
    return lookupMachinePlan(planIndex, row.MachineNo, row.ITControllerNo);
  }
  useEffect(() => {
    setPage(1);
  }, [search, pageSize, periodMode, periodAnchor]);
  useEffect(() => {
    let buffer = '';
    let flushTimer = null;
    function fireBuffered() {
      const code = buffer.trim();
      buffer = '';
      if (code.length >= 2 && !busyRef.current) fireRef.current(code);
    }
    function onKeydown(e) {
      if (busyRef.current) return;
      const tag = (e.target?.tagName || '').toLowerCase();
      if (tag === 'input' || tag === 'textarea' || tag === 'select') return;
      if (e.target?.closest?.('[data-scan-ignore]')) return;
      if (e.key === 'Enter') {
        if (flushTimer) clearTimeout(flushTimer);
        fireBuffered();
        return;
      }
      if (e.key && e.key.length === 1) {
        buffer += e.key;
        if (buffer.length >= 2) e.preventDefault();
        if (flushTimer) clearTimeout(flushTimer);
        flushTimer = setTimeout(fireBuffered, 120);
      }
    }
    window.addEventListener('keydown', onKeydown);
    return () => {
      window.removeEventListener('keydown', onKeydown);
      if (flushTimer) clearTimeout(flushTimer);
    };
  }, []);
  // ขั้นตอน MFG (ขั้นตอนเดียว): Scan Kanban → เทียบกับ Daily Plan → ตรง = บันทึกเป็น MATCHED
  async function runScanFlow(presetCode = '') {
    if (busyRef.current) return;
    busyRef.current = true;
    try {
      let qrCode = String(presetCode || '').trim();
      if (!looksLikeSpecQR(qrCode)) {
        qrCode = await scanStep({
          title: 'Scan Kanban',
          confirmText: 'บันทึก',
          validate: v => looksLikeSpecQR(v) ? undefined : 'ค่าที่สแกนไม่ใช่ QR ของ Specification sheet'
        });
        if (!qrCode) return;
        qrCode = qrCode.trim();
      }
      const machineNo = specQRMachineNo(qrCode);
      scanLoading('กำลังตรวจสอบกับ Daily Plan...');
      let spec = null;
      try {
        spec = await checkMFGSpecQR(qrCode);
      } catch (err) {
        scanClose();
        await scanErrorAlert(friendlyError(err, 'ตรวจสอบ QR ไม่สำเร็จ'));
        return;
      }
      scanClose();
      if (!spec || spec.state !== 'MATCH') {
        await scanSpecAlert(spec);
        return;
      }
      await submitScan({
        machineNo,
        qrCode
      });
    } finally {
      busyRef.current = false;
    }
  }
  function handleScannerFire(code) {
    if (busyRef.current) return;
    runScanFlow(code);
  }
  fireRef.current = handleScannerFire;
  async function submitScan({
    machineNo,
    partNo = '',
    serialNo = '',
    partType = '',
    qrCode = ''
  }) {
    setScanBusy(true);
    scanLoading('กำลังบันทึก...');
    let successMsg = '';
    try {
      const res = await scanMFGAssembly({
        machineNo,
        itControllerNo: partType === 'ITC' ? '' : serialNo,
        serialNo,
        partNo,
        partType,
        qrCode
      });
      const row = res?.row || {};
      const msg = res?.message || 'บันทึกแล้ว';
      const ok = res?.matched || res?.status === 'MATCHED';
      const isDuplicate = res?.duplicate || res?.status === 'DUPLICATE';
      const isRetired = res?.retiredFormat || res?.status === 'RETIRED_FORMAT';
      const isPartMismatch = Boolean(res?.partMismatch);
      if (row?.ID && !isDuplicate && !isRetired && !isPartMismatch) {
        await scanCloseWait();
        const itcLabel = row.ITControllerNo ? ` / No.: <b>${escapeHtml(row.ITControllerNo)}</b>` : '';
        const photoBlob = await scanPhotoCapture({
          title: 'ถ่ายรูปป้ายเครื่อง',
          html: `<div class="scan-popup-hint">Machine No: <b>${escapeHtml(machineNo || '-')}</b>${itcLabel}</div>`
        });
        if (photoBlob) {
          scanLoading('กำลังบันทึกรูป...');
          try {
            await uploadMFGAssemblyPhoto(row.ID, photoBlob);
          } catch (e) {
            scanClose();
            await scanErrorAlert('บันทึกรูปไม่สำเร็จ: ' + (e.message || ''));
          }
        }
      }
      scanClose();
      if (ok) {
        successMsg = msg;
      } else if (isPartMismatch) {
        await scanErrorAlert(res?.detail ? `${msg} — ${res.detail}` : msg);
      } else {
        // ผลสแกนที่ไม่ผ่านทุกกรณี (ไม่พบข้อมูล / ซ้ำ / รูปแบบเดิม / ต้องให้ WH สแกนก่อน)
        // แสดงเป็นหน้าต่างกลางจอแบบเดียวกับหน้า WH
        await scanErrorAlert(msg);
      }
    } catch (err) {
      scanClose();
      await scanErrorAlert(friendlyError(err, 'บันทึกไม่สำเร็จ'));
    } finally {
      setScanBusy(false);
      await loadRows();
    }
    if (successMsg) scanSuccessToast(successMsg);
  }
  async function applyPhotoUpload(id, fileOrBlob) {
    if (!id || photoBusy) return;
    setPhotoBusy(true);
    scanLoading('กำลังบันทึกรูป...');
    try {
      await uploadMFGAssemblyPhoto(id, fileOrBlob);
      scanClose();
      await scanSuccessToast('บันทึกรูปถ่ายแล้ว');
      await loadRows();
    } catch (err) {
      scanClose();
      await scanErrorAlert('บันทึกรูปไม่สำเร็จ: ' + (err.message || ''));
    } finally {
      setPhotoBusy(false);
    }
  }
  async function handleRetakePhoto(row) {
    if (!row?.ID || photoBusy) return;
    const photoBlob = await scanPhotoCapture({
      title: row.PhotoURL ? 'ถ่ายรูปป้ายใหม่' : 'ถ่ายรูปป้ายเครื่อง',
      html: `<div class="scan-popup-hint">Machine No: <b>${escapeHtml(row.MachineNo || '-')}</b>${row.ITControllerNo ? ` / No.: <b>${escapeHtml(row.ITControllerNo)}</b>` : ''}</div>`
    });
    if (!photoBlob) return;
    await applyPhotoUpload(row.ID, photoBlob);
  }
  function handleUploadPhotoClick(row) {
    if (!row?.ID || photoBusy) return;
    pendingPhotoRowIdRef.current = row.ID;
    photoFileInputRef.current?.click();
  }
  async function handleUploadPhotoChange(e) {
    const file = e.target.files?.[0];
    const targetId = pendingPhotoRowIdRef.current;
    e.target.value = '';
    pendingPhotoRowIdRef.current = null;
    if (!file || !targetId) return;
    await applyPhotoUpload(targetId, file);
  }
  async function runFieldScan(field) {
    const titles = {
      itControllerNo: 'No.',
      machineNo: 'Machine No',
      partNo: 'P/N',
      serialNo: 'S/N'
    };
    const code = await scanStep({
      title: titles[field] || field,
      confirmText: 'ใช้ค่านี้'
    });
    if (!code) return;
    const parsed = parseAssemblyCode(code);
    let val = code.trim();
    if (field === 'itControllerNo') val = parsed.itControllerNo || val;
    if (field === 'machineNo') val = parsed.machineNo || val;
    if (field === 'partNo' || field === 'serialNo') val = firstToken(val);
    setForm(f => ({
      ...f,
      [field]: val
    }));
  }
  function openAdd() {
    setEditId(null);
    setForm(EMPTY_FORM);
    setModalOpen(true);
  }
  function openEdit(row) {
    setEditId(row.ID);
    setForm({
      dateAssembly: toDateInput(row.DateAssembly),
      machineNo: row.MachineNo || '',
      itControllerNo: row.ITControllerNo || '',
      partNo: row.PartNo || '',
      serialNo: row.SerialNo || '',
      country: row.Country || '',
      checkDate: toDateInput(row.CheckDate),
      status: row.Status || ''
    });
    setModalOpen(true);
  }
  function closeModal() {
    if (saving) return;
    setModalOpen(false);
  }
  function setField(key, value) {
    setForm(f => ({
      ...f,
      [key]: value
    }));
  }
  async function save() {
    if (!form.machineNo.trim() || !form.itControllerNo.trim()) {
      toastError('กรุณากรอก Machine No และ IT Controller No.');
      return;
    }
    setSaving(true);
    try {
      if (editId) {
        await updateMFGAssembly(editId, form);
        toastSuccess('แก้ไขรายการแล้ว');
      } else {
        await createMFGAssembly(form);
        toastSuccess('เพิ่มรายการแล้ว');
      }
      setModalOpen(false);
      await loadRows();
    } catch (err) {
      toastError(friendlyError(err, 'บันทึกไม่สำเร็จ'));
    } finally {
      setSaving(false);
    }
  }
  async function handleDelete(row) {
    const label = row.MachineNo || row.ITControllerNo || '#' + row.ID;
    const ok = await confirmDelete({
      text: `ลบรายการ ${label}? กู้คืนไม่ได้`
    });
    if (!ok) return;
    try {
      await deleteMFGAssembly(row.ID);
      toastSuccess(`ลบรายการ ${label} แล้ว`);
      await loadRows();
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    }
  }
  const filtered = useMemo(() => {
    let list = rows.filter(r => (r.Status || '') === 'MATCHED');
    if (periodMode !== 'all') {
      list = list.filter(r => r.CheckDate && inPeriod(r.CheckDate, periodMode, periodAnchor));
    }
    const term = search.trim().toLowerCase();
    if (term) {
      list = list.filter(r => {
        const asm = assemblyFor(r);
        return [r.MachineNo, asm?.model, itDeviceOf(r, asm), r.Country, asm?.country, r.Status].some(v => String(v || '').toLowerCase().includes(term));
      });
    }
    return [...list].sort((a, b) => a.ID - b.ID);
  }, [rows, search, periodMode, periodAnchor, planIndex]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages));
  }
  return <AppShell navItems={MFG_NAV_ITEMS} roleLabel="MFG">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Matching Assembly</h2>
        </div>
      </div>

      <div className="pc-barcode-grid pc-barcode-grid--single">
        <div className="pc-barcode-card pc-card-mc" role="button" tabIndex={0} title="Scan Kanban" onClick={() => !scanBusy && runScanFlow()} onKeyDown={e => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          if (!scanBusy) runScanFlow();
        }
      }}>
          <span className="pc-barcode-kind">Kanban</span>
          <div className="pc-barcode-title">
            {scanBusy ? 'กำลังบันทึก...' : 'Part Confirmation'}
          </div>
          <div className="pc-barcode-box">
            <img className="pc-barcode-img" src={bcKanban} alt="บาร์โค้ด Kanban" />
          </div>
        </div>
      </div>

      <div className="prp-card">
        <PeriodRangePicker mode={periodMode} onModeChange={handlePeriodModeChange} anchor={periodAnchor} onAnchorChange={setPeriodAnchor} label="ช่วงวันที่ตรวจสอบ (Check Date)" countLabel={`${filtered.length} รายการ`} onClear={clearPeriod} />
      </div>

      {loadError && <p className="form-error" role="alert">
          {loadError}
        </p>}

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
        <div className="mfg-search-actions">
          <input className="wh-search" type="text" placeholder="ค้นหา Machine No / Model / IT device / Country / Status" value={search} onChange={e => setSearch(e.target.value)} />
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Item</th>
              <th>Date Ass'y</th>
              <th>Machine No</th>
              <th>Model</th>
              <th>IT device</th>
              <th>Country</th>
              <th>Check Date</th>
              <th>Check By</th>
              <th>รูปถ่าย</th>
              <th>Status</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={11} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>}
            {!loading && paged.map((a, idx) => {
            const meta = STATUS_META[a.Status] || {
              label: a.Status || '—',
              cls: 'il-badge il-badge-muted'
            };
            const asm = assemblyFor(a);
            const asmTitle = asm ? [asm.specCode && `Spec Code: ${asm.specCode}`, asm.specDetail && `Specification: ${asm.specDetail}`, asm.partsNumber && `Assembly Parts No.: ${asm.partsNumber}`, asm.itDevice && `IT device: ${asm.itDevice}`, asm.country && `ประเทศ: ${asm.country}`].filter(Boolean).join('\n') : '';
            return <tr key={a.ID}>
                    <td className="wh-cell-head" data-label="Item">
                      <strong>{(page - 1) * pageSize + idx + 1}</strong>
                    </td>
                    <td data-label="Date Ass'y">{fmtDate(a.DateAssembly)}</td>
                    <td className="il-mono" data-label="Machine No">
                      {a.MachineNo || '—'}
                    </td>
                    <td data-label="Model" title={asmTitle}>
                      {asm && asm.model ? <button type="button" className="mfg-model-link mfg-model-link-btn" onClick={() => setDetailRow({
                  row: a,
                  asm
                })} title="ดูรายละเอียดการประกอบ">
                          {asm.model}
                        </button> : '—'}
                    </td>
                    <td data-label="IT device">{itDeviceOf(a, asm) || '—'}</td>
                    <td data-label="Country">{a.Country || asm && asm.country || '—'}</td>
                    <td data-label="Check Date">{fmtDate(a.CheckDate)}</td>
                    <td data-label="Check By">{a.CreatedBy || '—'}</td>
                    <td data-label="รูปถ่าย">
                      {a.PhotoURL ? <button type="button" className="wh-photo-thumb" onClick={() => setPhotoView(a.PhotoURL)} title="คลิกเพื่อขยาย">
                          <img src={`${API_BASE_URL}${a.PhotoURL}`} alt="รูปถ่ายป้าย" loading="lazy" />
                        </button> : <span className="il-badge il-badge-muted">ไม่มีรูป</span>}
                    </td>
                    <td data-label="Status">
                      <span className={meta.cls} title={a.RetiredDetail || a.PlanDetail || a.PlanMessage || ''}>
                        {meta.label}
                      </span>
                    </td>
                    <td className="wh-cell-action">
                      {(asm || a.SpecDetail) && <button className="tsf-action-btn" onClick={() => setDetailRow({
                  row: a,
                  asm: asm || {}
                })} title="ดูรายละเอียดการประกอบ (ข้อมูล Spec sheet)">
                          รายละเอียด
                        </button>}
                      <button className="tsf-action-btn tsf-action-btn-warn" onClick={() => setPhotoEditRow(a)} disabled={photoBusy}>
                        แก้ไข
                      </button>
                      {a.Status !== 'MATCHED' && <button className="tsf-action-btn tsf-action-btn-danger" onClick={() => handleDelete(a)}>
                          ลบ
                        </button>}
                    </td>
                  </tr>;
          })}
            {!loading && filtered.length === 0 && <tr>
                <td colSpan={11} className="wh-empty-cell">
                  {rows.length === 0 ? 'ยังไม่มีรายการ' : 'ไม่พบรายการที่ค้นหา'}
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

      {modalOpen && <div className="wh-modal-overlay" onClick={closeModal}>
          <div className="wh-modal" onClick={e => e.stopPropagation()}>
            <h3 className="wh-modal-title">{editId ? 'แก้ไขรายการ' : 'เพิ่มรายการ'}</h3>

            <label className="wh-modal-label">Date Ass'y</label>
            <input className="wh-modal-input" type="date" value={form.dateAssembly} onChange={e => setField('dateAssembly', e.target.value)} />

            <label className="wh-modal-label">Machine No</label>
            <div style={{
          display: 'flex',
          gap: 8,
          alignItems: 'center'
        }}>
              <input className="wh-modal-input" style={{
            flex: 1
          }} value={form.machineNo} onChange={e => setField('machineNo', e.target.value)} placeholder="เช่น LX10400690" />
              <button type="button" className="tsf-action-btn" onClick={() => runFieldScan('machineNo')}>
                <QrCodeIcon className="size-4" /> สแกน
              </button>
            </div>

            <label className="wh-modal-label">No.</label>
            <div style={{
          display: 'flex',
          gap: 8,
          alignItems: 'center'
        }}>
              <input className="wh-modal-input" style={{
            flex: 1
          }} value={form.itControllerNo} onChange={e => setField('itControllerNo', e.target.value)} placeholder="เช่น 878250022802" />
              <button type="button" className="tsf-action-btn" onClick={() => runFieldScan('itControllerNo')}>
                <QrCodeIcon className="size-4" /> สแกน
              </button>
            </div>

            <label className="wh-modal-label">P/N</label>
            <div style={{
          display: 'flex',
          gap: 8,
          alignItems: 'center'
        }}>
              <input className="wh-modal-input" style={{
            flex: 1
          }} value={form.partNo} onChange={e => setField('partNo', e.target.value)} />
              <button type="button" className="tsf-action-btn" onClick={() => runFieldScan('partNo')}>
                <QrCodeIcon className="size-4" /> สแกน
              </button>
            </div>

            <label className="wh-modal-label">S/N</label>
            <div style={{
          display: 'flex',
          gap: 8,
          alignItems: 'center'
        }}>
              <input className="wh-modal-input" style={{
            flex: 1
          }} value={form.serialNo} onChange={e => setField('serialNo', e.target.value)} />
              <button type="button" className="tsf-action-btn" onClick={() => runFieldScan('serialNo')}>
                <QrCodeIcon className="size-4" /> สแกน
              </button>
            </div>

            <label className="wh-modal-label">Country</label>
            <input className="wh-modal-input" value={form.country} onChange={e => setField('country', e.target.value)} />

            <label className="wh-modal-label">Check Date</label>
            <input className="wh-modal-input" type="date" value={form.checkDate} onChange={e => setField('checkDate', e.target.value)} />

            <label className="wh-modal-label">Status</label>
            <SelectField value={form.status} onChange={v => setField('status', v)} options={[{
          value: '',
          label: '— ให้ระบบประเมินให้ —'
        }, ...STATUS_OPTIONS]} />

            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={closeModal} disabled={saving}>
                ยกเลิก
              </button>
              <button className="wh-modal-confirm" onClick={save} disabled={saving}>
                {saving ? 'กำลังบันทึก...' : 'บันทึก'}
              </button>
            </div>
          </div>
        </div>}

      {photoEditRow && <div className="wh-modal-overlay" onClick={() => setPhotoEditRow(null)}>
          <div className="wh-modal mfg-photo-modal" onClick={e => e.stopPropagation()}>
            <h3 className="wh-modal-title">
              {photoEditRow.PhotoURL ? 'แก้ไขรูป' : 'เพิ่มรูป'}
            </h3>

            <div className="mfg-photo-info">
              <div className="mfg-photo-info-row">
                <span className="mfg-photo-info-label">Machine No</span>
                <span className="mfg-photo-info-value">{photoEditRow.MachineNo || '—'}</span>
              </div>
              {photoEditRow.ITControllerNo ? <div className="mfg-photo-info-row">
                  <span className="mfg-photo-info-label">IT Controller</span>
                  <span className="mfg-photo-info-value">{photoEditRow.ITControllerNo}</span>
                </div> : null}
            </div>

            <div className="mfg-photo-choices">
              <button type="button" className="mfg-photo-choice" disabled={photoBusy} onClick={async () => {
            const row = photoEditRow;
            setPhotoEditRow(null);
            await handleRetakePhoto(row);
          }}>
                <CameraIcon className="size-5" />
                <span className="mfg-photo-choice-text">
                  <span className="mfg-photo-choice-title">ถ่ายรูปใหม่</span>
                </span>
              </button>
              <button type="button" className="mfg-photo-choice" disabled={photoBusy} onClick={() => {
            const row = photoEditRow;
            setPhotoEditRow(null);
            handleUploadPhotoClick(row);
          }}>
                <ArrowUpTrayIcon className="size-5" />
                <span className="mfg-photo-choice-text">
                  <span className="mfg-photo-choice-title">อัปโหลดรูป</span>
                </span>
              </button>
            </div>

            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={() => setPhotoEditRow(null)}>
                ยกเลิก
              </button>
            </div>
          </div>
        </div>}

      {detailRow && <div className="wh-modal-overlay" onClick={() => setDetailRow(null)}>
          <div className="wh-modal wh-detail-modal mfg-detail-wide" onClick={e => e.stopPropagation()}>
            <button type="button" className="wh-detail-close" onClick={() => setDetailRow(null)} aria-label="ปิด">
              <XMarkIcon className="size-4" />
            </button>

            <div className="wh-detail-header">
              <span className="wh-detail-header-icon">
                <WrenchScrewdriverIcon className="size-5" />
              </span>
              <div>
                <h3 className="wh-modal-title">รายละเอียดการประกอบ</h3>
                <span className="wh-detail-header-sub">{detailRow.asm.model || '—'}</span>
              </div>
            </div>

            <div className="wh-detail-section">
              <span className="wh-detail-section-title">
                <DocumentTextIcon className="size-4" /> ข้อมูลเครื่อง
              </span>
              <div className="wh-detail-grid">
                <div className="wh-detail-item">
                  <span className="wh-detail-label">Machine No</span>
                  <span className="wh-detail-value mono">{detailRow.row.MachineNo || '—'}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">Model</span>
                  <span className="wh-detail-value">{detailRow.asm.model || '—'}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">IT device</span>
                  <span className="wh-detail-value">{itDeviceOf(detailRow.row, detailRow.asm) || '—'}</span>
                </div>
                <div className="wh-detail-item">
                  <span className="wh-detail-label">Country</span>
                  <span className="wh-detail-value">
                    {detailRow.row.Country || detailRow.asm.country || '—'}
                  </span>
                </div>
              </div>
            </div>

            <div className="wh-detail-divider" />

            <div className="wh-detail-section">
              <span className="wh-detail-section-title">
                <CubeIcon className="size-4" /> ข้อมูล Spec sheet
              </span>
              {detailPlan.loading ? <span className="wh-detail-value">กำลังโหลด...</span> : detailPlan.data ? <div className="wh-detail-grid">
                  {detailPlan.columns.filter(col => col !== 'Machine' && String(detailPlan.data[col] ?? '').trim() !== '').map(col => <div className="wh-detail-item" key={col}>
                      <span className="wh-detail-label">{col.replace(/^\[\+\] /, '')}</span>
                      <span className="wh-detail-value">{detailPlan.data[col]}</span>
                    </div>)}
                </div> : <span className="wh-detail-value">ไม่พบเครื่องนี้ในตาราง Daily Plan</span>}
            </div>

            <div className="wh-detail-meta">
              <span>
                <TagIcon className="size-3.5" /> ประกอบโดย {detailRow.row.CreatedBy || '—'}
              </span>
              <span>
                <ClockIcon className="size-3.5" /> {fmtDate(detailRow.row.CheckDate)}
              </span>
            </div>

            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={() => setDetailRow(null)}>
                ปิด
              </button>
            </div>
          </div>
        </div>}

      {photoView && <div className="wh-modal-overlay" onClick={() => setPhotoView(null)}>
          <div className="wh-modal wh-photo-modal" onClick={e => e.stopPropagation()}>
            <h3 className="wh-modal-title">รูปถ่ายป้าย</h3>
            <div className="wh-photo-modal-img">
              <img src={`${API_BASE_URL}${photoView}`} alt="รูปถ่ายป้าย" />
            </div>
            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={() => setPhotoView(null)}>
                ปิด
              </button>
            </div>
          </div>
        </div>}

      <input ref={photoFileInputRef} type="file" accept="image/*" style={{
      display: 'none'
    }} onChange={handleUploadPhotoChange} />
    </AppShell>;
}
