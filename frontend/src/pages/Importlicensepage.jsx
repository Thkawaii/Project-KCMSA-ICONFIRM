import { useEffect, useMemo, useState } from 'react';
import { getImportLicenseItems, getImportLicenseSummary, uploadImportLicense, previewImportLicense, deleteImportLicenseItem, clearImportLicense, renewImportLicense } from '../api/importLicense.js';
import { getExportLicense, getExportLicenseTrace, uploadExportLicense, previewExportLicense, deleteExportLicense, clearExportLicense, renewExportLicense } from '../api/exportLicense.js';
import { PreviewResult, ChangePreview, ExtraColumnsCell } from '../components/FormatTools.jsx';
import AppShell from '../components/AppShell.jsx';
import FileDropZone from '../components/Filedropzone.jsx';
import SelectField from '../components/Selectfield.jsx';
import { confirmDelete, toastError, toastSuccess, promptRenewDays } from '../lib/toast.js';
import { computeLicenseExpiry, formatThaiDate, daysLeftLabel, STATUS_LABEL, EXPIRY_STATUS } from '../lib/licenseExpiry.js';
import { computeExportLicenseDates, exportLeadTimeDate, leadDaysLabel, leadBadgeClass, LEAD_STATUS, LEAD_STATUS_LABEL, LEAD_BADGE_CLASS, LEAD_FILTER_DUE_SOON, EXPORT_LICENSE_LEAD_DAYS, EXPORT_LICENSE_LEAD_WARN_DAYS } from '../lib/exportLicenseRules.js';
import { useDailyTick } from '../lib/useDailyTick.js';
import { useAppParams } from '../lib/nav.jsx';
import { buildStyledXlsxWorkbookBlob, downloadBlob } from '../lib/xlsx.js';
import PeriodRangePicker from '../components/PeriodRangePicker.jsx';
import { inPeriod, periodRangeLabel, periodFileTag } from '../lib/dateRange.js';
import { ChevronDoubleLeftIcon, ChevronDoubleRightIcon, ChevronLeftIcon, ChevronRightIcon, ClipboardDocumentCheckIcon, ClockIcon, CubeIcon, DocumentTextIcon, RectangleStackIcon, ReceiptPercentIcon, ShieldCheckIcon, Squares2X2Icon, TagIcon, TruckIcon, WrenchScrewdriverIcon, XMarkIcon } from '../components/icons.jsx';
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
  labelByRole: {
    LOG: 'Part Checklist'
  },
  icon: <ClipboardDocumentCheckIcon className="size-4" />,
  roles: ['WH', 'LOG']
}];
const EXPIRY_BADGE_CLASS = {
  [EXPIRY_STATUS.EXPIRED]: 'il-badge il-badge-bad',
  [EXPIRY_STATUS.EXPIRING]: 'il-badge il-badge-warn',
  [EXPIRY_STATUS.VALID]: 'il-badge il-badge-ok',
  [EXPIRY_STATUS.NO_DATE]: 'il-badge il-badge-muted'
};
function ExpiryCell({
  issueDate,
  expireDate
}) {
  const exp = expireDate ? computeExpireStatus(expireDate, 30) : computeLicenseExpiry(issueDate);
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
  const [modelFilter, setModelFilter] = useState('all');
  const [expiryFilter, setExpiryFilter] = useState('all');
  const [pageSize, setPageSize] = useState(25);
  const [page, setPage] = useState(1);
  const [detailRow, setDetailRow] = useState(null);
  const [file, setFile] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [uploadMsg, setUploadMsg] = useState(null);
  const [previewData, setPreviewData] = useState(null);
  const [previewing, setPreviewing] = useState(false);
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
  async function loadAll() {
    setLoading(true);
    setLoadError('');
    try {
      const [rows, sum] = await Promise.all([getImportLicenseItems(), getImportLicenseSummary()]);
      setItems(rows || []);
      setSummary(sum || []);
    } catch (err) {
      setLoadError(err.message || 'โหลดบัญชีใบอนุญาตนำเข้าไม่สำเร็จ');
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    loadAll();
  }, []);
  useEffect(() => {
    setPage(1);
  }, [selectedLot, search, modelFilter, expiryFilter, pageSize]);
  useEffect(() => {
    const lic = (params?.focusLicense || '').trim();
    if (!lic) return;
    setModelFilter('all');
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
      const result = await uploadImportLicense(file);
      setUploadMsg({
        success: `นำเข้าสำเร็จ — เพิ่มใหม่ ${result.imported} เครื่อง, อัปเดต ${result.updated} เครื่อง, ข้าม ${result.skipped} แถว`,
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
      const msg = err.message || 'ลบไม่สำเร็จ';
      setLoadError(msg);
      toastError(msg);
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
    const curLine = curExp?.hasDate ? `<div class="scan-popup-hint">วันหมดอายุปัจจุบัน: <b>${formatThaiDate(curExp.expiryDate)}</b> (${daysLeftLabel(curExp.daysLeft)})</div>` : '<div class="scan-popup-hint">ยังไม่ได้ระบุวันที่ออกใบอนุญาต — ต่ออายุจะนับวันหมดอายุใหม่จากวันนี้</div>';
    const days = await promptRenewDays({
      title: `ต่ออายุ ${label}`,
      html: `${curLine}<div class="scan-popup-hint">ระบบจะเลื่อนวันหมดอายุออกไปตามจำนวนวันที่กรอก</div>`,
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
    if (modelFilter !== 'all') {
      rows = rows.filter(r => (r.Model || '') === modelFilter);
    }
    if (expiryFilter !== 'all') {
      rows = rows.filter(r => computeLicenseExpiry(r.IssueDate).status === expiryFilter);
    }
    const term = search.trim().toLowerCase();
    if (term) {
      rows = rows.filter(r => (r.MachineNo || '').toLowerCase().includes(term) || (r.ProductionNo || '').toLowerCase().includes(term) || (r.LicenseNo || '').toLowerCase().includes(term) || (r.InvoiceNo || '').toLowerCase().includes(term) || (r.DeclarationNo || '').toLowerCase().includes(term) || (r.Model || '').toLowerCase().includes(term) || (r.ExportCountry || '').toLowerCase().includes(term));
    }
    rows = [...rows].sort((a, b) => {
      const da = a.IssueDate ? new Date(a.IssueDate).getTime() : NaN;
      const db = b.IssueDate ? new Date(b.IssueDate).getTime() : NaN;
      const va = Number.isNaN(da) ? -Infinity : da;
      const vb = Number.isNaN(db) ? -Infinity : db;
      return vb - va;
    });
    return rows;
  }, [items, selectedLot, modelFilter, expiryFilter, search, today]);
  const modelOptions = useMemo(() => {
    const set = new Set(items.map(r => r.Model).filter(Boolean));
    const list = Array.from(set).sort((a, b) => a.localeCompare(b));
    return [{
      value: 'all',
      label: 'ทุกแบบ/รุ่น'
    }, ...list.map(m => ({
      value: m,
      label: m
    }))];
  }, [items]);
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
  }], []);
  const lotOptions = useMemo(() => {
    const opts = [{
      value: '',
      label: 'ทุกใบอนุญาต'
    }];
    summary.forEach(s => {
      opts.push({
        value: `${s.LicenseNo}|${s.InvoiceNo}`,
        label: `${s.LicenseNo} · Invoice ${s.InvoiceNo} · ${s.Total} เครื่อง`
      });
    });
    return opts;
  }, [summary]);
  const counts = useMemo(() => ({
    total: items.length,
    licenses: new Set(items.map(r => r.LicenseNo).filter(Boolean)).size,
    invoices: new Set(items.map(r => r.InvoiceNo).filter(Boolean)).size,
    models: new Set(items.map(r => r.Model).filter(Boolean)).size
  }), [items]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), totalPages));
  }
  const currentLot = summary.find(s => `${s.LicenseNo}|${s.InvoiceNo}` === selectedLot);
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
        }} accept=".xlsx,.xls,.csv" label="อัปโหลดบัญชีใบอนุญาตนำเข้า" hint="ไฟล์ Excel หรือ CSV ที่มีคอลัมน์ หมายเลขเครื่อง / หมายเลขการผลิต / เลขใบอนุญาตนำเข้า / เลขอินวอยซ์นำเข้า" disabled={uploading} />
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

      <div className="dash-stats-row wh-stats-row">
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>เครื่องในบัญชีทั้งหมด</span>
            <span className="dash-stat-icon dash-icon-blue">
              <Squares2X2Icon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.total}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>ใบอนุญาตนำเข้า</span>
            <span className="dash-stat-icon dash-icon-red">
              <DocumentTextIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.licenses}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>อินวอยซ์นำเข้า</span>
            <span className="dash-stat-icon dash-icon-yellow">
              <ReceiptPercentIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.invoices}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>แบบ/รุ่น</span>
            <span className="dash-stat-icon dash-icon-green">
              <CubeIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{counts.models}</div>
        </div>
      </div>

      {summary.length > 0 && <div className="il-lot-filter">
          <label className="il-lot-filter-label">ใบอนุญาต</label>
          <div className="il-lot-filter-select">
            <SelectField value={selectedLot} onChange={setSelectedLot} options={lotOptions} />
          </div>
        </div>}

      {currentLot && <div className="wh-so-active-bar">
          <div>
            <span className="wh-so-active-label">ใบอนุญาตนำเข้า</span>
            <h3 className="wh-so-active-name">{currentLot.LicenseNo || '(ไม่มีเลขใบอนุญาต)'}</h3>
            <span className="wh-subtitle">
              Invoice {currentLot.InvoiceNo || '—'} · ใบขนสินค้า {currentLot.DeclarationNo || '—'} · รุ่น{' '}
              {currentLot.Model || '—'} · {currentLot.Total} เครื่อง
            </span>
          </div>
          <div className="il-lot-actions">
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
            <SelectField value={modelFilter} onChange={setModelFilter} options={modelOptions} />
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

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
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
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={15} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>}
            {!loading && paged.map((row, i) => <tr key={row.ID}>
                  <td className="wh-cell-head" data-label="ลำดับ">
                    {(page - 1) * pageSize + i + 1}
                  </td>
                  <td data-label="ตราอักษร">{row.Brand || '—'}</td>
                  <td data-label="แบบ/รุ่น">{row.Model || '—'}</td>
                  <td data-label="เลขใบอนุญาตนำเข้า">{row.LicenseNo || '—'}</td>
                  <td data-label="วันที่ออกใบอนุญาต">{formatThaiDate(row.IssueDate)}</td>
                  <td data-label="หมดอายุ (6 เดือน)">
                    <ExpiryCell issueDate={row.IssueDate} expireDate={row.ExpireDate} />
                  </td>
                  <td data-label="เลขอินวอยซ์นำเข้า">{row.InvoiceNo || '—'}</td>
                  <td data-label="เลขใบขนสินค้าขาเข้า">{row.DeclarationNo || '—'}</td>
                  <td data-label="จำนวน (เครื่อง)">{row.Qty}</td>
                  <td className="il-mono" data-label="หมายเลขเครื่อง">
                    <strong>{row.MachineNo}</strong>
                  </td>
                  <td className="il-mono" data-label="หมายเลขการผลิต">
                    {row.ProductionNo || '—'}
                  </td>
                  <td data-label="หมายเหตุ">{row.Remark || '—'}</td>
                  <td data-label="ส่งออกไปประเทศ">{row.ExportCountry || '—'}</td>
                  <td data-label="คอลัมน์เพิ่ม">
                    <ExtraColumnsCell json={row.extra_json} />
                  </td>
                  <td className="wh-cell-action">
                    <div className="il-row-actions">
                      <button className="wh-modal-cancel" onClick={() => setDetailRow(row)}>
                        รายละเอียด
                      </button>
                      <button className="wh-btn-danger" onClick={() => handleDeleteRow(row)}>
                        ลบ
                      </button>
                    </div>
                  </td>
                </tr>)}
            {!loading && paged.length === 0 && <tr>
                <td colSpan={15} className="wh-empty-cell">
                  ยังไม่มีข้อมูลในบัญชี — อัปโหลดไฟล์ Excel หรือ CSV ด้านบนก่อน
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

      {detailRow && <ImportDetailModal row={detailRow} onClose={() => setDetailRow(null)} />}
    </AppShell>;
}
function ImportDetailModal({
  row,
  onClose
}) {
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
  return <div className="wh-modal-overlay" onClick={onClose}>
      <div className="wh-modal wh-detail-modal" onClick={e => e.stopPropagation()}>
        <button type="button" className="wh-detail-close" onClick={onClose} aria-label="ปิด">
          <XMarkIcon className="size-4" />
        </button>

        <div className="wh-detail-header">
          <span className="wh-detail-header-icon">
            <DocumentTextIcon className="size-5" />
          </span>
          <div>
            <h3 className="wh-modal-title">รายละเอียดใบอนุญาตนำเข้า</h3>
            <span className="wh-detail-header-sub">{row.Model || row.Brand || '—'}</span>
          </div>
        </div>

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
            {item('ส่งออกไปประเทศ', row.ExportCountry)}
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
                <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
                {exp.hasDate && <span className="il-detail-days">{daysLeftLabel(exp.daysLeft)}</span>}
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

        <div className="wh-modal-actions">
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
// วันหมดอายุใบอนุญาตนำออก ยึด "วันที่นำออกใบอนุญาต + 1 เดือน" เสมอ
// ไม่ใช้วันหมดอายุที่ติดมากับไฟล์ Excel เพราะบางไฟล์ใส่มาไม่ตรงกติกา
// (เช่น นำออก 10 มี.ค. 2026 แต่ไฟล์ใส่หมดอายุ 31 ธ.ค. 2026 — ที่ถูกคือ 10 เม.ย. 2026)
function computeExportExpiry(row, withinDays = 7) {
  return computeExportLicenseDates(row, {
    withinDays
  });
}
function ExportOneMonthExpiryCell({
  row
}) {
  const exp = computeExportExpiry(row);
  return <div className="il-expiry-cell">
      <span className={EXPIRY_BADGE_CLASS[exp.status]}>{STATUS_LABEL[exp.status]}</span>
      {exp.hasDate && <>
          <span>{formatThaiDate(exp.expiryDate)}</span>
          <span className="il-expiry-days">{daysLeftLabel(exp.daysLeft)}</span>
        </>}
    </div>;
}
// Lead time — ต้องยื่นเรื่องให้ กสทช. ก่อนใบอนุญาตนำออกหมดอายุ 15 วัน
// สถานะมีแค่ 2 แบบ: "ถึงกำหนดยื่น" กับ "เลยกำหนดยื่น"
// ถ้าเป็น "ถึงกำหนดยื่น" แต่เหลือเวลาไม่เกิน 7 วัน ป้ายจะเปลี่ยนเป็นสีส้มเตือน (ข้อความเดิม)
function ExportLeadTimeCell({
  row
}) {
  const exp = computeExportExpiry(row);
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
function ExportTraceModal({
  row,
  country,
  onClose
}) {
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
  return <div className="wh-modal-overlay" onClick={onClose}>
      <div className="wh-modal wh-detail-modal" onClick={e => e.stopPropagation()}>
        <button type="button" className="wh-detail-close" onClick={onClose} aria-label="ปิด">
          <XMarkIcon className="size-4" />
        </button>

        <div className="wh-detail-header">
          <span className="wh-detail-header-icon">
            <TruckIcon className="size-5" />
          </span>
          <div>
            <h3 className="wh-modal-title">รายละเอียดใบอนุญาตส่งออก</h3>
            <span className="wh-detail-header-sub">{row.MachineNo || row.SerialNumber || '—'}</span>
          </div>
        </div>

        <div className="wh-detail-section">
          <span className="wh-detail-section-title">
            <CubeIcon className="size-4" /> ข้อมูลชิ้นงาน
          </span>
          <div className="wh-detail-grid">
            {item('Machine No', row.MachineNo)}
            {item('IT Controller S/N', row.ITControllerNo || row.SerialNumber)}
            {itemAlways('Serial Number', data?.masterData?.SerialNo)}
            {item('ประเทศปลายทาง', country)}
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
            {item('Export License', row.ExportLicenseNo || row.ExceptionLicense)}
            {itemAlways('Import License', row.ImportLicenseNo)}
            {item('วันที่นำออกใบอนุญาต', row.IssueDate ? formatThaiDate(row.IssueDate) : '')}
            <div className="wh-detail-item">
              <span className="wh-detail-label">หมดอายุ (1 เดือน)</span>
              <span className="wh-detail-value il-detail-status">
                <span className={EXPIRY_BADGE_CLASS[expiryInfo.status]}>{STATUS_LABEL[expiryInfo.status]}</span>
                {expiryInfo.hasDate && <span className="il-detail-days">{formatThaiDate(expiryInfo.expiryDate)} · {daysLeftLabel(expiryInfo.daysLeft)}</span>}
              </span>
            </div>
            <div className="wh-detail-item">
              <span className="wh-detail-label">{`Lead time (${EXPORT_LICENSE_LEAD_DAYS} วัน)`}</span>
              <span className="wh-detail-value il-detail-status">
                <span className={leadBadgeClass(expiryInfo)}>{LEAD_STATUS_LABEL[expiryInfo.leadStatus]}</span>
                {expiryInfo.hasDate && <span className="il-detail-days">{formatThaiDate(expiryInfo.leadDate)} · {leadDaysLabel(expiryInfo.leadDaysLeft)}</span>}
              </span>
            </div>
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
                  ไม่พบใบอนุญาตนำเข้าที่เชื่อมโยง — ตรวจสอบว่าเลข IT Controller (
                  {row.ITControllerNo || row.SerialNumber || '—'}) ตรงกับ “หมายเลขเครื่อง”
                  ในบัญชีใบอนุญาตนำเข้า และเป็นเลข 12 หลัก
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

        <div className="wh-modal-actions">
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
  const [exceptionFilter, setExceptionFilter] = useState('all');
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
    const b = String(r.SerialNumber || '').trim();
    return countryByITC[a] || countryByITC[b] || '';
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
      const UNKNOWN = 'ไม่ระบุประเทศ';
      const groups = new Map();
      list.forEach(r => {
        const key = countryOf(r) || UNKNOWN;
        if (!groups.has(key)) groups.set(key, []);
        groups.get(key).push(r);
      });
      const countryNames = Array.from(groups.keys()).sort((a, b) => {
        if (a === UNKNOWN) return 1;
        if (b === UNKNOWN) return -1;
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
        key: 'leadStatus',
        header: 'สถานะ Lead time',
        type: 'center',
        width: 22
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
          sheetName: country,
          columns: [...baseColumns, ...extraColumns],
          rows: list.map((r, i) => {
            const exp = computeExportExpiry(r);
            const extraValues = parseExtraJson(r.extra_json);
            const row = {
              __danger: exp.status === EXPIRY_STATUS.EXPIRED || exp.leadStatus === LEAD_STATUS.OVERDUE,
              item: i + 1,
              assemblyDate: r.AssemblyDate ? formatThaiDate(r.AssemblyDate) : '—',
              issueDate: r.IssueDate ? formatThaiDate(r.IssueDate) : '—',
              expiryDate: exp.hasDate ? formatThaiDate(exp.expiryDate) : '—',
              leadTimeDate: exp.hasDate ? formatThaiDate(exp.leadDate) : '—',
              leadStatus: exp.hasDate ? `${LEAD_STATUS_LABEL[exp.leadStatus]} (${leadDaysLabel(exp.leadDaysLeft)})` : '—',
              machineNo: dash2(r.MachineNo),
              itControllerNo: dash2(r.ITControllerNo || r.SerialNumber),
              invoiceNo: dash2(r.InvoiceNo),
              invoiceDate: r.InvoiceDate ? formatThaiDate(r.InvoiceDate) : '—',
              exportEntry: dash2(r.ExportEntry),
              importLicenseNo: dash2(r.ImportLicenseNo),
              exportLicenseNo: dash2(r.ExportLicenseNo || r.ExceptionLicense),
              country: country === UNKNOWN ? '—' : country,
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
  async function load() {
    setLoading(true);
    try {
      setRows(await getExportLicense());
    } catch (err) {
      toastError(err.message || 'โหลดบัญชีใบอนุญาตส่งออกไม่สำเร็จ');
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    load();
  }, []);
  async function handleRenewSelectedExport() {
    const licenseNo = exceptionFilter;
    if (!licenseNo || licenseNo === 'all') {
      toastError('กรุณาเลือกใบอนุญาตส่งออกที่ต้องการต่ออายุก่อน');
      return;
    }
    const days = await promptRenewDays({
      title: `ต่ออายุใบอนุญาตส่งออก ${licenseNo}`,
      html: '<div class="scan-popup-hint">ระบบจะเลื่อนวันหมดอายุออกไปตามจำนวนวันที่กรอก</div>',
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
  }, [search, exceptionFilter, expiryFilter, pageSize, periodMode, periodAnchor]);
  useEffect(() => {
    const exc = (params?.focusException || '').trim();
    const sn = (params?.focusSerial || '').trim();
    if (!exc && !sn) return;
    setExpiryFilter('all');
    setPeriodMode('all');
    setPeriodAnchor('');
    setExceptionFilter('all');
    setSearch(exc || sn);
  }, [params?.focusSerial, params?.focusException, params?.focusTs]);
  useEffect(() => {
    const exc = (params?.focusException || '').trim();
    if (!exc) return;
    if (rows.some(r => (r.ExceptionLicense || '') === exc)) {
      setExceptionFilter(exc);
      setSearch('');
    }
  }, [rows, params?.focusException, params?.focusTs]);
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
      const r = await uploadExportLicense(file);
      setMsg({
        success: `นำเข้าสำเร็จ — ${r.imported} แถว, ข้าม ${r.skipped} แถว`
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
      text: `ลบ Serial Number ${row.SerialNumber || '—'} ออกจากบัญชี?`
    });
    if (!ok) return;
    try {
      await deleteExportLicense(row.ID);
      await load();
      toastSuccess(`ลบ ${row.SerialNumber || ''} แล้ว`);
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
    const licenseNo = exceptionFilter;
    if (!licenseNo || licenseNo === 'all') return;
    const ok = await confirmDelete({
      text: `ลบใบอนุญาตส่งออก ${licenseNo} ออกจากระบบทั้งใบ? กู้คืนไม่ได้`,
      confirmText: 'ลบทั้งใบ'
    });
    if (!ok) return;
    try {
      const res = await clearExportLicense(licenseNo);
      setExceptionFilter('all');
      await load();
      toastSuccess(`ลบใบอนุญาตส่งออก ${licenseNo} แล้ว (${res?.deleted ?? 0} รายการ)`);
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    }
  }
  const filtered = useMemo(() => {
    let list = rows;
    if (exceptionFilter !== 'all') {
      list = list.filter(r => (r.ExceptionLicense || '') === exceptionFilter);
    }
    const term = search.trim().toLowerCase();
    if (term) {
      list = list.filter(r => [r.SerialNumber, r.ExceptionLicense, r.MachineNo, r.ITControllerNo, r.InvoiceNo, r.ExportEntry, r.ImportLicenseNo, r.ExportLicenseNo, countryOf(r)].filter(Boolean).some(v => String(v).toLowerCase().includes(term)));
    }
    if (periodMode !== 'all') {
      list = list.filter(r => r.AssemblyDate && inPeriod(r.AssemblyDate, periodMode, periodAnchor));
    }
    if (expiryFilter !== 'all') {
      list = list.filter(r => {
        const exp = computeExportExpiry(r);
        if (expiryFilter === LEAD_FILTER_DUE_SOON) return exp.hasDate && exp.leadUrgent;
        if (expiryFilter === LEAD_STATUS.OVERDUE || expiryFilter === LEAD_STATUS.DUE) {
          return exp.leadStatus === expiryFilter;
        }
        return exp.status === expiryFilter;
      });
    }
    return list;
  }, [rows, exceptionFilter, expiryFilter, search, countryByITC, periodMode, periodAnchor]);
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
  const exceptionOptions = useMemo(() => {
    const set = new Set(rows.map(r => r.ExceptionLicense).filter(Boolean));
    const list = Array.from(set).sort((a, b) => a.localeCompare(b));
    return [{
      value: 'all',
      label: 'Export License(ทุกใบ)'
    }, ...list.map(m => ({
      value: m,
      label: m
    }))];
  }, [rows]);
  const expiryOptions = useMemo(() => [{
    value: 'all',
    label: 'ทุกสถานะ'
  }, {
    value: LEAD_STATUS.OVERDUE,
    label: `Lead time · ${LEAD_STATUS_LABEL[LEAD_STATUS.OVERDUE]}`
  }, {
    value: LEAD_FILTER_DUE_SOON,
    label: `Lead time · ใกล้ครบกำหนด (≤ ${EXPORT_LICENSE_LEAD_WARN_DAYS} วัน)`
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
  }], []);
  const currentLicenseRows = useMemo(() => {
    if (exceptionFilter === 'all') return [];
    return rows.filter(r => (r.ExceptionLicense || '') === exceptionFilter);
  }, [rows, exceptionFilter]);
  const currentLicenseInvoices = useMemo(() => {
    const set = new Set(currentLicenseRows.map(r => r.InvoiceNo).filter(Boolean));
    return Array.from(set).sort((a, b) => a.localeCompare(b));
  }, [currentLicenseRows]);
  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);
  return <>
      <div className="wh-upload-card">
        <div className="fdz-row">
          <FileDropZone file={file} onSelect={f => {
          setFile(f);
          setMsg(null);
          setPreviewData(null);
        }} accept=".xlsx,.xls,.csv" label="อัปโหลดบัญชีใบอนุญาตส่งออก" hint="ไฟล์ Excel หรือ CSV ที่มีคอลัมน์ ใบขน (Date) / Exception License / Serial Number / Expire date (อัปโหลดซ้ำ Serial เดิม ระบบทับให้)" disabled={uploading} />
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
      </div>

      <div className="il-export-filter-card">
        <PeriodRangePicker mode={periodMode} onModeChange={handlePeriodModeChange} anchor={periodAnchor} onAnchorChange={setPeriodAnchor} min={asmDateBounds.min} max={asmDateBounds.max} label="เลือกช่วงวันที่สำหรับ Export แยกประเทศ" countLabel={`${filtered.length} รายการ`} />
        {periodMode !== 'all' && <button type="button" className="qa-clear-btn" onClick={() => {
        setPeriodMode('all');
        setPeriodAnchor('');
      }}>
            <XMarkIcon className="size-4" />
            ล้างช่วง
          </button>}
      </div>

      {rows.length > 0 && <div className="il-lot-filter">
          <label className="il-lot-filter-label">ใบอนุญาตส่งออก</label>
          <div className="il-lot-filter-select">
            <SelectField value={exceptionFilter} onChange={setExceptionFilter} options={exceptionOptions} />
          </div>
        </div>}

      {exceptionFilter !== 'all' && <div className="wh-so-active-bar">
          <div>
            <span className="wh-so-active-label">ใบอนุญาตส่งออก</span>
            <h3 className="wh-so-active-name">{exceptionFilter || '(ไม่มีเลขใบอนุญาต)'}</h3>
            <span className="wh-subtitle">
              Invoice {currentLicenseInvoices.length > 0 ? currentLicenseInvoices.join(', ') : '—'} ·{' '}
              {currentLicenseRows.length} เครื่อง
            </span>
          </div>
          <div className="il-lot-actions">
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
        <div className="il-filter-search-group">
          <div className="wh-pagesize-select il-model-filter">
            <SelectField value={expiryFilter} onChange={setExpiryFilter} options={expiryOptions} />
          </div>
          <input className="wh-search" type="text" placeholder="ค้นหา Machine No / IT Controller / Invoice / License / ประเทศ" value={search} onChange={e => setSearch(e.target.value)} />
          <button className="wh-issue-btn" onClick={handleExportByCountry} disabled={exportingXlsx || rows.length === 0} title={`ดาวน์โหลด Excel แยกชีตตามประเทศปลายทาง — ช่วง ${periodLabel}`}>
            {exportingXlsx ? 'กำลัง Export...' : 'Export Excel (แยกประเทศ)'}
          </button>
          {rows.length > 0 && <button className="wh-btn-danger" onClick={handleClearAll}>
              ลบทุกใบอนุญาต
            </button>}
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
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
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={15} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>}
            {!loading && paged.map((row, i) => <tr key={row.ID}>
                  <td className="wh-cell-head" data-label="Item">
                    {(page - 1) * pageSize + i + 1}
                  </td>
                  <td data-label="Date Ass'y">{formatThaiDate(row.AssemblyDate)}</td>
                  <td className="il-mono wh-cell-head" data-label="Machine No">
                    <strong>{row.MachineNo || '—'}</strong>
                  </td>
                  <td className="il-mono" data-label="IT Controller S/N">
                    {row.ITControllerNo || row.SerialNumber || '—'}
                  </td>
                  <td data-label="Country">{countryOf(row) || '—'}</td>
                  <td data-label="Invoice">
                    <div className="il-mono">{row.InvoiceNo || '—'}</div>
                    {row.InvoiceDate && <div className="il-invoice-date">
                        {formatThaiDate(row.InvoiceDate)}
                      </div>}
                  </td>
                  <td className="il-mono" data-label="Export Entry">
                    {row.ExportEntry || '—'}
                  </td>
                  <td className="il-mono" data-label="Import License">
                    {row.ImportLicenseNo || '—'}
                  </td>
                  <td className="il-mono" data-label="Export License">
                    {row.ExportLicenseNo || row.ExceptionLicense || '—'}
                  </td>
                  <td data-label="วันที่นำออกใบอนุญาต">
                    {formatThaiDate(row.IssueDate)}
                  </td>
                  <td data-label="หมดอายุ (1 เดือน)">
                    <ExportOneMonthExpiryCell row={row} />
                  </td>
                  <td data-label={`Lead time (${EXPORT_LICENSE_LEAD_DAYS} วัน)`}>
                    <ExportLeadTimeCell row={row} />
                  </td>
                  <td data-label="Remark">{row.Remark || '—'}</td>
                  <td data-label="คอลัมน์เพิ่ม">
                    <ExtraColumnsCell json={row.extra_json} />
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
                </tr>)}
            {!loading && paged.length === 0 && <tr>
                <td colSpan={15} className="wh-empty-cell">
                  ยังไม่มีข้อมูลใบอนุญาตส่งออก — อัปโหลดไฟล์ Excel หรือ CSV ด้านบนก่อน
                </td>
              </tr>}
          </tbody>
        </table>
      </div>

      {traceRow && <ExportTraceModal row={traceRow} country={countryOf(traceRow)} onClose={() => setTraceRow(null)} />}

      {!loading && filtered.length > pageSize && <div className="tsf-pagination">
          <span className="wh-subtitle" style={{
        fontSize: 13
      }}>
            Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, filtered.length)} of{' '}
            {filtered.length} entries
          </span>
          <div className="tsf-pagination-buttons">
            <button className="wh-modal-cancel" onClick={() => setPage(Math.max(1, page - 1))} disabled={page === 1}>
              <ChevronLeftIcon className="size-4" />
            </button>
            <span className="tsf-pagination-current">
              {page} / {totalPages}
            </span>
            <button className="wh-modal-cancel" onClick={() => setPage(Math.min(totalPages, page + 1))} disabled={page === totalPages}>
              <ChevronRightIcon className="size-4" />
            </button>
          </div>
        </div>}
    </>;
}