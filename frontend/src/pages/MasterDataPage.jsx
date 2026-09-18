import { useEffect, useMemo, useRef, useState } from 'react';
import AppShell from '../components/AppShell.jsx';
import SelectField from '../components/Selectfield.jsx';
import useFileDrop from '../lib/useFileDrop.js';
import { getMasterData, uploadMasterData, deleteMasterData, clearMasterData, previewMasterDataChanges, updateMasterData } from '../api/masterData.js';
import { getUploadData, uploadDataFile, deleteUploadDataRow, updateUploadDataRow, clearUploadData, previewUploadData } from '../api/uploadData.js';
import { PreviewResult, ChangePreview } from '../components/FormatTools.jsx';
import { EditableCell, EditHint, LockBadge, useLiveRefresh } from '../components/InlineEdit.jsx';
import { ADMIN_NAV_ITEMS } from './AdminDashboardpage.jsx';
import { confirmUploadDeletes, syncResultParts } from '../lib/uploadSync.js';
import { confirmDelete, toastError, toastSuccess } from '../lib/toast.js';
import { buildStyledXlsxBlob, buildStyledXlsxWorkbookBlob, downloadBlob } from '../lib/xlsx.js';
import { CloudArrowUpIcon } from '../components/icons.jsx';
import { ArrowDownTrayIcon, CpuChipIcon, CubeIcon, RectangleStackIcon, Squares2X2Icon, TagIcon } from '../components/icons.jsx';
import '../UploadData.css';
const COMPONENT_TYPES = [{
  value: 'it_controller',
  label: 'IT Controller',
  noLabel: 'IT Controller no.'
}, {
  value: 'swing_motor',
  label: 'Swing Motor',
  noLabel: 'Swing Motor No.'
}, {
  value: 'pump_assy_hyd',
  label: 'Pump Assy HYD',
  noLabel: 'Pump Assy HYD NO.'
}, {
  value: 'motor_propel',
  label: 'Motor Propel',
  noLabel: 'Motor Propel NO.'
}, {
  value: 'control_valve',
  label: 'Control Valve',
  noLabel: 'Control Valve NO.'
}];
const COMPONENT_TYPE_VALUES = new Set(COMPONENT_TYPES.map(t => t.value));
// ALL PART: อัปโหลดไฟล์เดียวได้ทุก Part — backend อ่านทุกชีต แล้วแยกชนิดให้เองจาก
// คอลัมน์ Type / ชื่อชีต / หัวคอลัมน์เฉพาะชนิด (เช่น "Swing Motor No.")
const ALL_PARTS_UPLOAD = 'all';
function isMasterUploadType(value) {
  return value === ALL_PARTS_UPLOAD || COMPONENT_TYPE_VALUES.has(value);
}
const DATASET_TYPES = [{
  value: 'planning',
  label: 'Planning'
}, {
  // Daily Plan: ไฟล์ Daily plan ที่มีชีต "Specification sheet" (เครื่องละ 5 แถว) — ใช้เทียบกับ QR ที่ MFG สแกน
  value: 'daily_plan',
  label: 'Daily Plan'
}, {
  value: 'wh1',
  label: 'WH1'
}, {
  value: 'wh2',
  label: 'WH2'
}, {
  value: 'engine',
  label: 'Engine'
}];
const UPLOAD_TYPE_OPTIONS = [{
  value: ALL_PARTS_UPLOAD,
  label: 'ALL PART'
}, ...COMPONENT_TYPES.map(t => ({
  value: t.value,
  label: t.label
})), ...DATASET_TYPES];
const TYPE_OPTIONS = [{
  value: 'it_controller',
  label: 'ALL PART'
}, ...DATASET_TYPES];
const ALL_TYPE_LABELS = Object.fromEntries([{
  value: ALL_PARTS_UPLOAD,
  label: 'ALL PART'
}, ...COMPONENT_TYPES, ...DATASET_TYPES].map(t => [t.value, t.label]));
function formatByType(byType) {
  if (!Array.isArray(byType) || byType.length === 0) return '';
  return byType.map(t => `${typeLabel(t.component_type)} ${t.count}`).join(' · ');
}
function typeLabel(value) {
  return ALL_TYPE_LABELS[value] || value;
}
const COMPONENT_TYPE_FILTER = [{
  value: 'all',
  label: 'ทุกชนิด'
}, ...COMPONENT_TYPES.map(t => ({
  value: t.value,
  label: t.label
}))];
const NO_LABEL_BY_TYPE = {
  all: 'No.',
  ...Object.fromEntries(COMPONENT_TYPES.map(t => [t.value, t.noLabel]))
};
const HEADING_LABEL_BY_TYPE = {
  all: 'ALL PART',
  ...Object.fromEntries(COMPONENT_TYPES.map(t => [t.value, t.label]))
};
// ชื่อตรงกับ "IT device" ใน Daily Plan
const CONNECTIVITY_LABELS = {
  SATELLITE_IRIDIUM: 'IT(Satellite, iridium)',
  MOBILE_4G_HIGH_H: 'IT(Mobile4G, high speed-H)',
  MOBILE_4G_NORMAL: 'IT(Mobile4G, normal speed)',
  MOBILE_4G3_HIGH: 'IT(Mobile4G-3, high speed)',
  MOBILE_4G_HIGH: 'IT(Mobile4G, high speed)',
  UNKNOWN: 'ไม่ระบุ'
};
const CONNECTIVITY_ORDER = ['SATELLITE_IRIDIUM', 'MOBILE_4G_HIGH_H', 'MOBILE_4G_NORMAL', 'MOBILE_4G3_HIGH', 'MOBILE_4G_HIGH', 'UNKNOWN'];
const CONNECTIVITY_FILTER = [{
  value: 'all',
  label: 'ทุก Connectivity'
}, ...CONNECTIVITY_ORDER.map(v => ({
  value: v,
  label: CONNECTIVITY_LABELS[v]
}))];
const uploadNavItems = [{
  to: '/master-data',
  label: 'อัพโหลดข้อมูล',
  icon: <RectangleStackIcon className="size-4" />
}];
const DASH = '—';
export default function MasterDataPage() {
  const role = (localStorage.getItem('iconfirm_role') || '').toUpperCase();
  // แก้ไขในตาราง (ปุ่มดินสอ) เฉพาะ ADMIN — UPLOAD แก้ข้อมูลใน Excel แล้วอัปโหลดใหม่
  const canEdit = role === 'ADMIN';
  const navItems = canEdit ? ADMIN_NAV_ITEMS : uploadNavItems;
  const shellRoleLabel = canEdit ? 'Admin' : 'Upload';
  const [uploadType, setUploadType] = useState(ALL_PARTS_UPLOAD);
  const [viewType, setViewType] = useState('it_controller');
  const [compType, setCompType] = useState('all');
  const [pendingFile, setPendingFile] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [uploadMsg, setUploadMsg] = useState(null);
  const [previewData, setPreviewData] = useState(null);
  const [previewing, setPreviewing] = useState(false);
  const fileInputRef = useRef(null);
  const isMasterType = isMasterUploadType(uploadType);
  const isAllParts = uploadType === ALL_PARTS_UPLOAD;
  const canPreview = true;
  async function handlePreview() {
    if (!pendingFile) {
      setUploadMsg({
        error: 'กรุณาเลือกไฟล์ก่อนตรวจสอบ'
      });
      return;
    }
    setPreviewing(true);
    setPreviewData(null);
    try {
      const data = isMasterType ? await previewMasterDataChanges(pendingFile, uploadType) : await previewUploadData(uploadType, pendingFile);
      const useChangeView = isMasterType || !!data?.summary;
      setPreviewData({
        ...data,
        _mode: useChangeView ? 'change' : 'map'
      });
    } catch (err) {
      setUploadMsg({
        error: err.message || 'ตรวจสอบไฟล์ไม่สำเร็จ'
      });
    } finally {
      setPreviewing(false);
    }
  }
  const [reloadKey, setReloadKey] = useState(0);
  function acceptFile(file) {
    setPendingFile(file || null);
    setUploadMsg(null);
    setPreviewData(null);
  }
  function handleFileChange(e) {
    acceptFile(e.target.files?.[0] || null);
  }
  const {
    dragging: fileDragging,
    stateClass: fileDropState,
    dropProps: fileDropProps
  } = useFileDrop({
    accept: '.xlsx,.xls,.csv',
    disabled: uploading,
    onFile: acceptFile,
    onReject: (file, hint) => setUploadMsg({
      error: `ไฟล์ "${file.name}" ไม่รองรับ — ต้องเป็น ${hint}`
    })
  });
  async function handleUpload() {
    if (!pendingFile) {
      setUploadMsg({
        error: 'กรุณาเลือกไฟล์ Excel ก่อน'
      });
      return;
    }
    setUploading(true);
    setUploadMsg(null);
    try {
      let summary = previewData?.summary;
      if (!summary) {
        const pv = isMasterType ? await previewMasterDataChanges(pendingFile, uploadType) : await previewUploadData(uploadType, pendingFile);
        summary = pv?.summary;
      }
      if (!(await confirmUploadDeletes(summary, pendingFile.name))) {
        return;
      }
      if (isMasterType) {
        const result = await uploadMasterData(pendingFile, uploadType);
        const breakdown = isAllParts ? formatByType(result.byType) : '';
        setUploadMsg({
          success: [`เพิ่มใหม่ ${result.imported ?? 0}`, `อัปเดต ${result.updated ?? 0}`, ...(result.unchanged ? [`เหมือนเดิม ${result.unchanged}`] : []), ...(result.locked ? [`สแกนแล้ว ไม่อัปเดต ${result.locked}`] : []), ...syncResultParts(result)].join(' · ') + (breakdown ? ` — ${breakdown}` : ''),
          problems: result.problems || []
        });
      } else {
        const result = await uploadDataFile(uploadType, pendingFile);
        const parts = [`เพิ่มใหม่ ${result.imported ?? 0}`, `อัปเดต ${result.updated ?? 0}`];
        if (result.duplicate) parts.push(`เหมือนเดิม ${result.duplicate}`);
        if (result.locked) parts.push(`สแกนแล้ว ไม่อัปเดต ${result.locked}`);
        parts.push(...syncResultParts(result));
        if (result.skipped) parts.push(`ข้าม ${result.skipped}`);
        setUploadMsg({
          success: parts.join(' · '),
          problems: result.problems || []
        });
      }
      setPendingFile(null);
      setPreviewData(null);
      if (fileInputRef.current) fileInputRef.current.value = '';
      if (isMasterType) {
        setViewType('it_controller');
        setCompType(uploadType === 'it_controller' || isAllParts ? 'all' : uploadType);
      } else {
        setViewType(uploadType);
      }
      setReloadKey(n => n + 1);
    } catch (err) {
      setUploadMsg({
        error: err.message || 'อัปโหลดไม่สำเร็จ'
      });
    } finally {
      setUploading(false);
    }
  }
  return <AppShell navItems={navItems} roleLabel={shellRoleLabel}>
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">อัพโหลดข้อมูล</h2>
        </div>
      </div>

      <div className="upload-panel upload-panel-wide">
        <label className={['upload-dropzone', 'upload-panel-dropzone', pendingFile ? 'upload-dropzone-filled' : '', fileDropState].filter(Boolean).join(' ')} htmlFor="md-file" {...fileDropProps}>
          <input id="md-file" ref={fileInputRef} type="file" accept=".xlsx,.xls,.csv" onChange={handleFileChange} className="upload-card-input-hidden" />
          <span className="upload-dropzone-icon">
            <CloudArrowUpIcon className="size-[26px]" />
          </span>
          <span className="upload-dropzone-text">
            {fileDragging ? <span className="dz-drop-text">
                <span className="dz-arrow">↓</span> ปล่อยไฟล์ได้เลย
              </span> : pendingFile ? pendingFile.name : typeLabel(uploadType)}
          </span>
          <span className="upload-dropzone-hint">.xlsx, .xls, .csv</span>
        </label>

        <div className="upload-panel-side">
          <div className="upload-panel-field">
            <label className="upload-panel-label" htmlFor="md-upload-type">
              ประเภทที่อัปโหลด
            </label>
            <SelectField value={uploadType} onChange={v => {
            setUploadType(v);
            setPendingFile(null);
            setUploadMsg(null);
            setPreviewData(null);
            if (fileInputRef.current) fileInputRef.current.value = '';
          }} options={UPLOAD_TYPE_OPTIONS.map(t => ({
            value: t.value,
            label: t.label
          }))} />
          </div>

          <div style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 8
        }}>
            {canPreview && <button className="qa-fail-btn upload-panel-btn" style={{
            background: '#eef2ff',
            color: '#4338ca',
            borderColor: '#c7d2fe'
          }} disabled={previewing || uploading} onClick={handlePreview}>
                {previewing ? 'กำลังตรวจสอบ...' : 'ตรวจสอบก่อนอัปโหลด'}
              </button>}
            <button className="wh-issue-btn upload-panel-btn" disabled={uploading} onClick={handleUpload}>
              {uploading ? 'กำลังอัปโหลด...' : `อัปโหลด ${typeLabel(uploadType)}`}
            </button>
          </div>
        </div>

        {previewData && (previewData._mode === 'change' ? <ChangePreview result={previewData} typeLabel={typeLabel} /> : <PreviewResult result={previewData} />)}

        {uploadMsg?.success && <p className="upload-card-msg upload-card-msg-ok">{uploadMsg.success}</p>}
        {uploadMsg?.error && <p className="upload-card-msg upload-card-msg-err">{uploadMsg.error}</p>}

        {uploadMsg?.problems?.length > 0 && <ul className="upload-card-msg upload-card-msg-err" style={{
        textAlign: 'left',
        margin: '8px 0 0'
      }}>
            {uploadMsg.problems.map((problem, i) => <li key={i}>{problem}</li>)}
          </ul>}
      </div>

      <div className="wh-heading-row" style={{
      marginTop: 28
    }}>
        <div>
          <h2 className="wh-title" style={{
          fontSize: 19
        }}>
            รายการ — {viewType === 'it_controller' ? HEADING_LABEL_BY_TYPE[compType] : typeLabel(viewType)}
          </h2>
        </div>
        <div className="md-type-field">
          <SelectField value={viewType} onChange={setViewType} options={TYPE_OPTIONS.map(t => ({
          value: t.value,
          label: t.label
        }))} />
        </div>
      </div>

      {viewType === 'it_controller' ? <ITControllerView canEdit={canEdit} reloadKey={reloadKey} bumpReload={() => setReloadKey(n => n + 1)} compType={compType} setCompType={setCompType} /> : <DatasetView key={`${viewType}-${reloadKey}`} dataset={viewType} canEdit={canEdit} />}
    </AppShell>;
}
function ITControllerView({
  canEdit = false,
  reloadKey,
  bumpReload,
  compType,
  setCompType
}) {
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [keyword, setKeyword] = useState('');
  const [deletingId, setDeletingId] = useState(0);
  const [connFilter, setConnFilter] = useState('all');
  const noLabel = NO_LABEL_BY_TYPE[compType] || 'IT Controller no.';
  const showITCols = compType === 'all' || compType === 'it_controller';
  const loadSeq = useRef(0);
  async function load(silent = false) {
    const seq = ++loadSeq.current;
    if (!silent) {
      setLoading(true);
      setLoadError('');
    }
    try {
      const data = await getMasterData({});
      if (seq === loadSeq.current) setRows(data || []);
    } catch (err) {
      if (!silent && seq === loadSeq.current) setLoadError(err.message || 'โหลดทะเบียนไม่สำเร็จ');
    } finally {
      if (!silent && seq === loadSeq.current) setLoading(false);
    }
  }
  useEffect(() => {
    load(false);
  }, [reloadKey]);
  useLiveRefresh(() => load(true));
  async function saveField(row, key, value) {
    try {
      const updated = await updateMasterData(row.ID, {
        [key]: value
      });
      setRows(list => list.map(r => r.ID === row.ID ? {
        ...r,
        ...updated
      } : r));
    } catch (err) {
      if (err?.status === 409) load(true);
      throw err;
    }
  }
  const filtered = useMemo(() => {
    let result = rows;
    if (compType !== 'all') {
      result = result.filter(row => row.ComponentType === compType);
    }
    const kw = keyword.trim().toLowerCase();
    if (kw) {
      result = result.filter(row => [row.Name, row.Model, row.PartNo, row.SerialNo, row.ITControllerNo, row.IMEI].filter(Boolean).some(field => String(field).toLowerCase().includes(kw)));
    }
    if (showITCols && connFilter !== 'all') {
      result = result.filter(row => (row.ConnectivityType || 'UNKNOWN') === connFilter);
    }
    return result;
  }, [rows, keyword, compType, connFilter, showITCols]);
  const stats = useMemo(() => {
    const withImei = rows.filter(row => row.IMEI).length;
    const models = new Set(rows.map(row => row.Model).filter(Boolean));
    const partNos = new Set(rows.map(row => row.PartNo).filter(Boolean));
    return {
      total: rows.length,
      withImei,
      models: models.size,
      partNos: partNos.size
    };
  }, [rows]);
  async function handleDelete(row) {
    const label = row.SerialNo || row.Name || `รายการ #${row.ID}`;
    const ok = await confirmDelete({
      text: `ลบ ${label} ออกจากทะเบียน? กู้คืนไม่ได้`
    });
    if (!ok) return;
    setDeletingId(row.ID);
    setLoadError('');
    try {
      await deleteMasterData(row.ID);
      bumpReload();
      toastSuccess(`ลบ ${label} แล้ว`);
    } catch (err) {
      const msg = err.message || 'ลบไม่สำเร็จ';
      toastError(msg);
      if (err?.status === 409) load(true);
    } finally {
      setDeletingId(0);
    }
  }
  async function handleClearAll() {
    const isAll = compType === 'all';
    const label = isAll ? 'ทะเบียนทั้งหมด' : HEADING_LABEL_BY_TYPE[compType] || compType;
    const ok = await confirmDelete({
      text: `ลบ ${label} ออกจากทะเบียนทั้งหมด? กู้คืนไม่ได้`,
      confirmText: 'ลบทั้งหมด'
    });
    if (!ok) return;
    setLoadError('');
    try {
      const res = isAll ? await clearMasterData({
        all: true
      }) : await clearMasterData({
        componentType: compType
      });
      bumpReload();
      toastSuccess(`ลบแล้ว ${res.deleted ?? 0} รายการ`);
    } catch (err) {
      const msg = err.message || 'ลบไม่สำเร็จ';
      setLoadError(msg);
      toastError(msg);
    }
  }
  function handleExport() {
    const columns = showITCols ? [{
      key: 'itemNo',
      header: 'Item No.',
      type: 'number',
      width: 8
    }, {
      key: 'name',
      header: 'Part Name',
      type: 'text'
    }, {
      key: 'model',
      header: 'Model',
      type: 'text'
    }, {
      key: 'partNo',
      header: 'Part No.',
      type: 'text'
    }, {
      key: 'serialNo',
      header: 'Serial No.',
      type: 'text'
    }, {
      key: 'itcNo',
      header: noLabel,
      type: 'text'
    }, {
      key: 'imei',
      header: 'IMEI',
      type: 'text'
    }, {
      key: 'connectivity',
      header: 'Connectivity',
      type: 'text'
    }] : [{
      key: 'itemNo',
      header: 'Item No.',
      type: 'number',
      width: 8
    }, {
      key: 'name',
      header: 'Part Name',
      type: 'text'
    }, {
      key: 'model',
      header: 'Model',
      type: 'text'
    }, {
      key: 'serialNo',
      header: 'Serial No.',
      type: 'text'
    }, {
      key: 'itcNo',
      header: noLabel,
      type: 'text'
    }];
    if (compType === 'all') {
      // ใส่ชนิดอะไหล่ไว้ในไฟล์ เพื่อให้แก้แล้วอัปโหลดกลับด้วย ALL PART ได้ทันที
      columns.splice(1, 0, {
        key: 'partType',
        header: 'Part Type',
        type: 'text'
      });
    }
    const toRows = list => list.map((row, i) => ({
      itemNo: i + 1,
      partType: typeLabel(row.ComponentType),
      name: row.Name || '',
      model: row.Model || '',
      partNo: row.PartNo || '',
      serialNo: row.SerialNo || '',
      itcNo: row.ITControllerNo || '',
      imei: row.IMEI || '',
      connectivity: row.ComponentType === 'it_controller' ? CONNECTIVITY_LABELS[row.ConnectivityType || 'UNKNOWN'] : ''
    }));
    const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-');
    if (compType === 'all') {
      const groups = CONNECTIVITY_ORDER.map(code => ({
        sheetName: CONNECTIVITY_LABELS[code].slice(0, 31),
        columns,
        rows: toRows(filtered.filter(row => (row.ConnectivityType || 'UNKNOWN') === code))
      })).filter(g => g.rows.length > 0);
      if (groups.length === 0) return;
      const blob = buildStyledXlsxWorkbookBlob({
        sheets: groups
      });
      downloadBlob(blob, `master-data-all-part-${stamp}.xlsx`);
      return;
    }
    const sheetName = (HEADING_LABEL_BY_TYPE[compType] || 'Master Data').slice(0, 31);
    const blob = buildStyledXlsxBlob({
      sheetName,
      columns,
      rows: toRows(filtered)
    });
    downloadBlob(blob, `master-data-${compType}-${stamp}.xlsx`);
  }
  const colCount = showITCols ? 10 : 7;
  const lockedCount = filtered.filter(r => r.Locked).length;
  return <>
      {loadError && <p className="form-error" role="alert">
          {loadError}
        </p>}

      <div className="dash-stats-row wh-stats-row" style={{
      marginTop: 4,
      marginBottom: 24
    }}>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>รายการทั้งหมด</span>
            <span className="dash-stat-icon dash-icon-blue">
              <Squares2X2Icon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{stats.total}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>มี IMEI</span>
            <span className="dash-stat-icon dash-icon-green">
              <CpuChipIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{stats.withImei}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>จำนวน Model</span>
            <span className="dash-stat-icon dash-icon-yellow">
              <TagIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{stats.models}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>จำนวน Part No.</span>
            <span className="dash-stat-icon dash-icon-red">
              <CubeIcon className="size-4" />
            </span>
          </div>
          <div className="dash-stat-value">{stats.partNos}</div>
        </div>
      </div>

      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{
          fontSize: 17
        }}>
            {(keyword.trim() || compType !== 'all') && `พบ ${filtered.length} จาก ${rows.length}`}
          </h2>
          <EditHint lockedCount={lockedCount} canEdit={canEdit} />
        </div>
        <div className="uv-list-tools md-list-tools" style={{
        display: 'flex',
        gap: 10
      }}>
          <div className="md-type-field" style={{
          minWidth: 170
        }}>
            <SelectField value={compType} onChange={setCompType} options={COMPONENT_TYPE_FILTER.map(t => ({
            value: t.value,
            label: t.label
          }))} />
          </div>
          {showITCols && <div className="md-type-field" style={{
          minWidth: 190
        }}>
              <SelectField value={connFilter} onChange={setConnFilter} options={CONNECTIVITY_FILTER} />
            </div>}
          <input className="wh-search" type="search" value={keyword} onChange={e => setKeyword(e.target.value)} placeholder={showITCols ? `สแกนหรือพิมพ์ S/N, ${noLabel}, IMEI, P/N` : `สแกนหรือพิมพ์ S/N, ${noLabel}`} style={{
          minWidth: 200,
          flex: '1 1 200px'
        }} />
          <button className="wh-issue-btn" onClick={handleExport} disabled={filtered.length === 0}>
            <ArrowDownTrayIcon className="size-4" /> Export Excel
          </button>
          <button className="qa-fail-btn" onClick={handleClearAll} disabled={rows.length === 0}>
            {compType === 'all' ? 'ลบทั้งหมด' : 'ลบทั้งชนิด'}
          </button>
        </div>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Item No.</th>
              <th>Part Name</th>
              <th>Model</th>
              {showITCols && <th>Part No.</th>}
              <th>Serial No.</th>
              <th>{noLabel}</th>
              {showITCols && <th>IMEI</th>}
              {showITCols && <th>Connectivity</th>}
              <th>สถานะการสแกน</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={colCount} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>}

            {!loading && filtered.map((row, i) => {
            const isITC = row.ComponentType === 'it_controller';
            const lock = {
              locked: !!row.Locked,
              lockReason: row.LockReason,
              readOnly: !canEdit
            };
            return <tr key={row.ID} className={row.Locked ? 'ie-row-locked' : ''}>
                  <td className="wh-cell-head" data-label="Item No.">
                    <strong>{i + 1}</strong>
                  </td>
                  <td data-label="Part Name">
                    <EditableCell {...lock} label="Part Name" value={row.Name} onSave={v => saveField(row, 'Name', v)} />
                  </td>
                  <td data-label="Model">
                    <EditableCell {...lock} label="Model" value={row.Model} onSave={v => saveField(row, 'Model', v)} />
                  </td>
                  {showITCols && <td data-label="Part No." style={codeStyle}>
                      {isITC ? <EditableCell {...lock} mono label="Part No." value={row.PartNo} onSave={v => saveField(row, 'PartNo', v)} /> : DASH}
                    </td>}
                  <td data-label="Serial No." style={codeStyle}>
                    <EditableCell {...lock} mono label="Serial No." value={row.SerialNo} onSave={v => saveField(row, 'SerialNo', v)} />
                  </td>
                  <td data-label={noLabel} style={codeStyle}>
                    <EditableCell {...lock} mono label={NO_LABEL_BY_TYPE[row.ComponentType] || noLabel} value={row.ITControllerNo} onSave={v => saveField(row, 'ITControllerNo', v)} />
                  </td>
                  {showITCols && <td data-label="IMEI" style={codeStyle}>
                      {isITC ? <EditableCell {...lock} mono label="IMEI" value={row.IMEI} onSave={v => saveField(row, 'IMEI', v)} /> : DASH}
                    </td>}
                  {showITCols && <td data-label="Connectivity">
                      {isITC ? <EditableCell {...lock} type="select" label="Connectivity" value={row.ConnectivityType || ''} options={CONNECTIVITY_EDIT_OPTIONS} display={CONNECTIVITY_LABELS[row.ConnectivityType || 'UNKNOWN']} onSave={v => saveField(row, 'ConnectivityType', v)} /> : DASH}
                    </td>}
                  <td data-label="สถานะการสแกน">
                    {row.Locked ? <LockBadge locked reason={row.LockReason} /> : <span className="ie-empty">ยังไม่ได้สแกน</span>}
                  </td>
                  <td className="wh-cell-action">
                    <div style={{
                  display: 'flex',
                  gap: 6,
                  justifyContent: 'flex-end'
                }}>
                      <button className="qa-fail-btn" disabled={deletingId === row.ID || row.Locked} title={row.Locked ? `ลบไม่ได้ — ${row.LockReason || 'สแกนผ่านแล้ว'}` : ''} onClick={() => handleDelete(row)}>
                        {deletingId === row.ID ? 'กำลังลบ...' : 'ลบ'}
                      </button>
                    </div>
                  </td>
                </tr>;
          })}

            {!loading && filtered.length === 0 && <tr>
                <td colSpan={colCount} className="wh-empty-cell">
                  {keyword.trim() || compType !== 'all' ? 'ไม่พบรายการตามตัวกรอง' : 'ยังไม่มีข้อมูลในทะเบียน'}
                </td>
              </tr>}
          </tbody>
        </table>
      </div>
    </>;
}
const CONNECTIVITY_EDIT_OPTIONS = [...CONNECTIVITY_ORDER.filter(v => v !== 'UNKNOWN').map(v => ({
  value: v,
  label: CONNECTIVITY_LABELS[v]
})), {
  value: '',
  label: 'อัตโนมัติ',
  hint: 'ระบบจัดให้ตาม Part Name / Model'
}];

const UD_PAGE_SIZE = 100;
function DatasetView({
  dataset,
  canEdit = false
}) {
  const label = typeLabel(dataset);
  const [columns, setColumns] = useState([]);
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [keyword, setKeyword] = useState('');
  const [exporting, setExporting] = useState(false);
  const [localReload, setLocalReload] = useState(0);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const loadSeq = useRef(0);
  function runSearch() {
    setPage(1);
    setLocalReload(n => n + 1);
  }
  const firstKeywordRun = useRef(true);
  useEffect(() => {
    if (firstKeywordRun.current) {
      firstKeywordRun.current = false;
      return;
    }
    const t = setTimeout(runSearch, 350);
    return () => clearTimeout(t);
  }, [keyword]);
  async function load(silent = false) {
    const seq = ++loadSeq.current;
    if (!silent) {
      setLoading(true);
      setLoadError('');
    }
    try {
      const data = await getUploadData(dataset, keyword || undefined, page, UD_PAGE_SIZE);
      if (seq !== loadSeq.current) return;
      setColumns(data?.columns || []);
      setRows(data?.rows || []);
      setTotal(data?.total ?? 0);
      setTotalPages(data?.totalPages || 1);
    } catch (err) {
      if (!silent && seq === loadSeq.current) setLoadError(err.message || 'โหลดรายการไม่สำเร็จ');
    } finally {
      if (!silent && seq === loadSeq.current) setLoading(false);
    }
  }
  useEffect(() => {
    load(false);
  }, [dataset, localReload, page]);
  useLiveRefresh(() => load(true));
  async function saveCell(row, col, value) {
    try {
      const res = await updateUploadDataRow(row.ID, {
        [col]: value
      });
      if (res?.row) {
        setRows(list => list.map(r => r.ID === row.ID ? res.row : r));
      }
    } catch (err) {
      if (err?.status === 409) load(true);
      throw err;
    }
  }
  async function handleDelete(row) {
    const ok = await confirmDelete({
      text: 'ลบแถวนี้? กู้คืนไม่ได้'
    });
    if (!ok) return;
    try {
      await deleteUploadDataRow(row.ID);
      setLocalReload(n => n + 1);
      toastSuccess('ลบแถวแล้ว');
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
      if (err?.status === 409) load(true);
    }
  }
  async function handleClear() {
    const ok = await confirmDelete({
      text: `ล้างข้อมูล ${label} ทั้งหมด? กู้คืนไม่ได้`
    });
    if (!ok) return;
    try {
      const res = await clearUploadData(dataset);
      setPage(1);
      setLocalReload(n => n + 1);
      toastSuccess(`ล้างแล้ว ${res.deleted ?? 0} แถว`);
    } catch (err) {
      toastError(err.message || 'ล้างไม่สำเร็จ');
    }
  }
  async function handleExport() {
    setExporting(true);
    setLoadError('');
    try {
      const PAGE = 500;
      let all = [];
      let cols = [];
      let p = 1;
      for (let guard = 0; guard < 2000; guard++) {
        const data = await getUploadData(dataset, undefined, p, PAGE);
        if (p === 1) cols = data?.columns || [];
        const batch = data?.rows || [];
        all = all.concat(batch);
        const totalPages = data?.totalPages || 1;
        if (p >= totalPages || batch.length === 0) break;
        p += 1;
      }
      if (all.length === 0) {
        setLoadError('ยังไม่มีข้อมูลให้ Export');
        return;
      }
      const parsed = all.map(row => {
        try {
          return JSON.parse(row.DataJSON || '{}');
        } catch {
          return {};
        }
      });
      const isPlanningExport = dataset === 'planning';
      const exportCols = isPlanningExport ? cols.filter(label => label !== 'Line') : cols;
      const numericByCol = exportCols.map(label => {
        // LOT NO. (เช่น 10.040) ต้องคงเป็นข้อความ ไม่งั้นเลข 0 ท้ายหาย
        if (label === 'LOT NO.') return false;
        let sawValue = false;
        for (const obj of parsed) {
          const raw = obj[label];
          if (raw == null || String(raw).trim() === '') continue;
          sawValue = true;
          const s = String(raw).trim().replace(/,/g, '');
          if (!/^-?\d+(\.\d+)?$/.test(s)) return false;
          if (s.length > 11 || /^0\d/.test(s)) return false;
        }
        return sawValue;
      });
      let noNumeric = false;
      if (isPlanningExport) {
        let sawValue = false;
        noNumeric = true;
        for (const obj of parsed) {
          const raw = obj['Line'];
          if (raw == null || String(raw).trim() === '') continue;
          sawValue = true;
          const s = String(raw).trim().replace(/,/g, '');
          if (!/^-?\d+(\.\d+)?$/.test(s)) {
            noNumeric = false;
            break;
          }
        }
        noNumeric = sawValue && noNumeric;
      }
      const columns = [{
        key: '_no',
        header: '#',
        type: isPlanningExport ? noNumeric ? 'number' : 'text' : 'number',
        width: 6
      }, ...exportCols.map((label, i) => ({
        key: `c${i}`,
        header: label,
        type: numericByCol[i] ? 'number' : 'text'
      }))];
      const rows = parsed.map((obj, idx) => {
        const fallbackNo = idx + 1;
        let noVal = fallbackNo;
        if (isPlanningExport) {
          const raw = obj['Line'];
          const s = raw == null ? '' : String(raw).trim();
          if (s !== '') {
            noVal = noNumeric ? Number(s.replace(/,/g, '')) : s;
          }
        }
        const out = {
          _no: noVal
        };
        exportCols.forEach((label, i) => {
          const raw = obj[label];
          if (numericByCol[i]) {
            const s = raw == null ? '' : String(raw).trim().replace(/,/g, '');
            out[`c${i}`] = s === '' ? null : Number(s);
          } else {
            out[`c${i}`] = raw == null || String(raw).trim() === '' ? '' : String(raw);
          }
        });
        return out;
      });
      const sheetName = (label || 'Data').slice(0, 31);
      const blob = buildStyledXlsxBlob({
        sheetName,
        columns,
        rows
      });
      const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-');
      downloadBlob(blob, `${dataset}-export-${stamp}.xlsx`);
    } catch (err) {
      setLoadError(err.message || 'Export ไม่สำเร็จ');
    } finally {
      setExporting(false);
    }
  }
  function cellValue(row, colName) {
    try {
      const data = JSON.parse(row.DataJSON || '{}');
      return data[colName] ?? '';
    } catch {
      return '';
    }
  }
  const lockedCount = rows.filter(r => r.Locked).length;
  const isPlanning = dataset === 'planning';
  const displayColumns = isPlanning ? columns.filter(c => c !== 'Line') : columns;
  return <>
      {loadError && <p className="form-error" role="alert">
          {loadError}
        </p>}

      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title" style={{
          fontSize: 17
        }}>
            {total.toLocaleString()} รายการ
          </h2>
          <EditHint lockedCount={lockedCount} canEdit={canEdit} />
        </div>
        <div className="uv-list-tools" style={{
        display: 'flex',
        gap: 10,
        flexWrap: 'wrap'
      }}>
          <input className="wh-search" placeholder="ค้นหา (เลขเครื่อง / LOT / Order / Parts)" value={keyword} onChange={e => setKeyword(e.target.value)} style={{
          minWidth: 240
        }} />
          <button className="wh-issue-btn" onClick={handleExport} disabled={exporting}>
            {exporting ? 'กำลัง Export...' : <>
                <ArrowDownTrayIcon className="size-4" /> Export Excel
              </>}
          </button>
          <button className="qa-fail-btn" onClick={handleClear} disabled={total === 0}>
            ล้างทั้งหมด
          </button>
        </div>
      </div>

      <div className="wh-table-card ud-table-scroll">
        <table className="wh-table ud-table">
          <thead>
            <tr>
              <th className="ud-th-sticky">#</th>
              {displayColumns.map(c => <th key={c}>{c}</th>)}
              <th>สถานะการสแกน</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={displayColumns.length + 3} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>}
            {!loading && rows.map((row, i) => <tr key={row.ID} className={row.Locked ? 'ie-row-locked' : ''}>
                  <td className="ud-td-sticky">{isPlanning ? cellValue(row, 'Line') || (page - 1) * UD_PAGE_SIZE + i + 1 : (page - 1) * UD_PAGE_SIZE + i + 1}</td>
                  {displayColumns.map(c => <td key={c} data-label={c}>
                      <EditableCell locked={!!row.Locked} lockReason={row.LockReason} readOnly={!canEdit} label={c} value={cellValue(row, c)} onSave={v => saveCell(row, c, v)} />
                    </td>)}
                  <td data-label="สถานะการสแกน">
                    {row.Locked ? <LockBadge locked reason={row.LockReason} /> : <span className="ie-empty">ยังไม่ได้สแกน</span>}
                  </td>
                  <td className="wh-cell-action">
                    <div style={{
                display: 'flex',
                gap: 6,
                justifyContent: 'flex-end'
              }}>
                      <button className="qa-fail-btn" disabled={row.Locked} title={row.Locked ? `ลบไม่ได้ — ${row.LockReason || 'สแกนผ่านแล้ว'}` : ''} onClick={() => handleDelete(row)}>
                        ลบ
                      </button>
                    </div>
                  </td>
                </tr>)}
            {!loading && rows.length === 0 && <tr>
                <td colSpan={displayColumns.length + 3} className="wh-empty-cell">
                  ยังไม่มีรายการที่อัปโหลด
                </td>
              </tr>}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && <div className="ud-pager" style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'flex-end',
      gap: 12,
      marginTop: 12
    }}>
          <button className="wh-issue-btn" onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1 || loading}>
            ก่อนหน้า
          </button>
          <span style={{
        fontSize: 14
      }}>
            หน้า {page.toLocaleString()} / {totalPages.toLocaleString()}
          </span>
          <button className="wh-issue-btn" onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page >= totalPages || loading}>
            ถัดไป
          </button>
        </div>}

    </>;
}
const codeStyle = {
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace'
};
