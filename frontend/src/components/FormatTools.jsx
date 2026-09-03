import { useEffect, useState } from 'react';
import SelectField from './Selectfield.jsx';
import './FormatTools.css';
import { confirmDelete, toastError, toastSuccess } from '../lib/toast.js';
import { getColumnAliases, createColumnAlias, deleteColumnAlias, getCodeAliases, createCodeAlias, deleteCodeAlias, uploadCodeAliases } from '../api/formatConfig.js';
import { updateMasterData } from '../api/masterData.js';
import { buildStyledXlsxBlob, downloadBlob } from '../lib/xlsx.js';
import useFileDrop from '../lib/useFileDrop.js';
const panelStyle = {
  border: '1px solid #e2e8f0',
  borderRadius: 12,
  background: '#fff',
  marginTop: 16,
  overflow: 'hidden'
};
const panelHeadStyle = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 10,
  padding: '12px 16px',
  cursor: 'pointer',
  background: '#f8fafc',
  fontWeight: 600,
  fontSize: 14
};
const panelBodyStyle = {
  padding: '14px 16px'
};
const codeStyle = {
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace'
};
function Collapsible({
  title,
  hint,
  children,
  defaultOpen = false
}) {
  const [open, setOpen] = useState(defaultOpen);
  return <div style={panelStyle}>
      <div style={panelHeadStyle} onClick={() => setOpen(v => !v)}>
        <span>
          {title}
          {hint && <span style={{
          fontWeight: 400,
          color: '#94a3b8',
          marginLeft: 8
        }}>{hint}</span>}
        </span>
        <span style={{
        color: '#64748b'
      }}>{open ? '▲' : '▼'}</span>
      </div>
      {open && <div style={panelBodyStyle}>{children}</div>}
    </div>;
}
export function ColumnAliasPanel({
  scope,
  targetOptions = [],
  embedded = false
}) {
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(false);
  const [source, setSource] = useState('');
  const [target, setTarget] = useState('');
  const [note, setNote] = useState('');
  const [changeKind, setChangeKind] = useState('rename');
  const [saving, setSaving] = useState(false);
  const hasTargets = Array.isArray(targetOptions) && targetOptions.length > 0;
  async function load() {
    setLoading(true);
    try {
      const data = await getColumnAliases(scope);
      setRows(Array.isArray(data) ? data : []);
    } catch (err) {
      toastError(err.message || 'โหลดรายการจับคู่คอลัมน์ไม่สำเร็จ');
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    load();
    setSource('');
    setTarget('');
    setNote('');
  }, [scope]);
  async function handleAdd() {
    if (!source.trim()) {
      toastError('กรอกชื่อหัวคอลัมน์');
      return;
    }
    const isAdd = changeKind === 'add';
    if (!isAdd && !target.trim()) {
      toastError('เลือก "ข้อมูลเดิม" ที่จะให้แม็ปไปหา');
      return;
    }
    const finalTarget = isAdd ? source.trim() : target.trim();
    setSaving(true);
    try {
      await createColumnAlias({
        table: scope,
        new: source.trim(),
        old: finalTarget,
        note: note.trim(),
        kind: changeKind
      });
      setSource('');
      setTarget('');
      setNote('');
      toastSuccess('เพิ่มการจับคู่คอลัมน์แล้ว');
      await load();
    } catch (err) {
      toastError(err.message || 'เพิ่มไม่สำเร็จ');
    } finally {
      setSaving(false);
    }
  }
  async function handleDelete(id) {
    const ok = await confirmDelete({
      text: 'ลบการจับคู่คอลัมน์นี้?'
    });
    if (!ok) return;
    try {
      await deleteColumnAlias(id);
      toastSuccess('ลบแล้ว');
      await load();
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    }
  }
  const CHANGE_KIND_LABEL = {
    rename: 'เปลี่ยนชื่อ',
    add: 'เพิ่มใหม่',
    reorder: 'สลับตำแหน่ง'
  };
  const sourceLabel = changeKind === 'add' ? 'ชื่อหัวคอลัมน์ที่เพิ่มเข้ามา' : 'ชื่อหัวคอลัมน์ใหม่ (ที่เปลี่ยนมา)';
  const body = <>
      <div className="fmt-field" style={{
      maxWidth: 260,
      marginBottom: 12
    }}>
        <label className="fmt-label">ชนิดการเปลี่ยน</label>
        <SelectField value={changeKind} onChange={setChangeKind} options={[{
        value: 'rename',
        label: 'เปลี่ยนชื่อหัวคอลัมน์'
      }, {
        value: 'add',
        label: 'เพิ่มหัวคอลัมน์ใหม่'
      }, {
        value: 'reorder',
        label: 'สลับตำแหน่งคอลัมน์'
      }]} />
      </div>

      {changeKind === 'reorder' ? <div style={{
      background: '#f0fdfa',
      border: '1px solid #99f6e4',
      borderRadius: 10,
      padding: '12px 14px',
      fontSize: 13,
      color: '#0f766e',
      lineHeight: 1.6
    }}>
          การ <b>สลับตำแหน่งคอลัมน์</b> ระบบรองรับให้อัตโนมัติอยู่แล้ว — เพราะจับคู่ด้วย
          <b> ชื่อหัวคอลัมน์</b> ไม่ใช่ตำแหน่ง ดังนั้นย้ายคอลัมน์ไปไว้ตรงไหนก็อ่านถูก
          <b> ไม่ต้องตั้งค่าเพิ่ม</b>
        </div> : <>
          <div className="fmt-form">
            <div className="fmt-field">
              <label className="fmt-label">{sourceLabel}</label>
              <input className="fmt-input" value={source} onChange={e => setSource(e.target.value)} placeholder="เช่น หมายเลขเครื่อง (ใหม่)" />
            </div>
            {changeKind !== 'add' && <div className="fmt-field">
                <label className="fmt-label">ข้อมูลเดิม</label>
                {hasTargets ? <SelectField value={target} onChange={setTarget} options={targetOptions.map(t => typeof t === 'string' ? {
            value: t,
            label: t
          } : {
            value: t.value,
            label: t.label
          })} placeholder="— เลือกคอลัมน์ —" /> : <input className="fmt-input" value={target} onChange={e => setTarget(e.target.value)} placeholder="เช่น Machine" />}
              </div>}
            <div className="fmt-field">
              <label className="fmt-label">หมายเหตุ (ไม่บังคับ)</label>
              <input className="fmt-input" value={note} onChange={e => setNote(e.target.value)} />
            </div>
          </div>

          {changeKind === 'add' && <p style={{
        fontSize: 12.5,
        color: '#64748b',
        margin: '8px 2px 0'
      }}>
              เพิ่มหัวคอลัมน์ใหม่ไม่ต้องเลือก "ข้อมูลเดิม" — ระบบจะเก็บคอลัมน์นี้เป็น "คอลัมน์เพิ่ม" ให้เอง
            </p>}

          <div className="fmt-actions">
            <button className="wh-issue-btn fmt-add-btn" onClick={handleAdd} disabled={saving}>
              {saving ? 'กำลังเพิ่ม...' : 'เพิ่ม'}
            </button>
          </div>
        </>}

      <div className="fmt-table-wrap">
        <table className="wh-table" style={{
        width: '100%'
      }}>
          <thead>
            <tr>
              <th>ชนิด</th>
              <th>ข้อมูลใหม่ที่จะเปลี่ยน</th>
              <th>→ ข้อมูลเดิม</th>
              <th>หมายเหตุ</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={5} className="wh-empty-cell">กำลังโหลด...</td>
              </tr>}
            {!loading && rows.length === 0 && <tr>
                <td colSpan={5} className="wh-empty-cell">ยังไม่มีการจับคู่ — ไฟล์ปกติไม่ต้องตั้งค่าอะไร</td>
              </tr>}
            {!loading && rows.map(r => <tr key={r.id}>
                  <td>{CHANGE_KIND_LABEL[r.kind] || 'เปลี่ยนชื่อ'}</td>
                  <td style={codeStyle}>{r.new}</td>
                  <td style={codeStyle}>{r.old}</td>
                  <td>{r.note || '—'}</td>
                  <td className="wh-cell-action">
                    <button className="qa-fail-btn" onClick={() => handleDelete(r.id)}>ลบ</button>
                  </td>
                </tr>)}
          </tbody>
        </table>
      </div>
    </>;
  if (embedded) return body;
  return <Collapsible title="จับคู่หัวคอลัมน์ (เมื่อไฟล์เปลี่ยนชื่อหัวคอลัมน์)" defaultOpen>
      {body}
    </Collapsible>;
}
const KIND_LABEL = {
  machine: 'Machine No.',
  pn: 'P/N',
  sn: 'S/N'
};
export function CodeAliasPanel({
  componentType = 'it_controller',
  embedded = false
}) {
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(false);
  const [fromCode, setFromCode] = useState('');
  const [toSerial, setToSerial] = useState('');
  const [kind, setKind] = useState('machine');
  const [note, setNote] = useState('');
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const kindText = KIND_LABEL[kind] || '';
  async function load() {
    setLoading(true);
    try {
      const data = await getCodeAliases({
        componentType
      });
      setRows(Array.isArray(data) ? data : []);
    } catch (err) {
      toastError(err.message || 'โหลดรายการจับคู่ค่ารหัสไม่สำเร็จ');
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    load();
  }, [componentType]);
  async function handleAdd() {
    if (!fromCode.trim() || !toSerial.trim()) {
      toastError('กรอกทั้ง "New (ค่าใหม่)" และ "Old (ค่าเดิม)"');
      return;
    }
    setSaving(true);
    try {
      await createCodeAlias({
        new: fromCode.trim(),
        to_serial_no: toSerial.trim(),
        to_part_no: '',
        component_type: componentType,
        kind,
        note: note.trim()
      });
      setFromCode('');
      setToSerial('');
      setNote('');
      toastSuccess('เพิ่มการจับคู่ค่ารหัสแล้ว');
      await load();
    } catch (err) {
      toastError(err.message || 'เพิ่มไม่สำเร็จ');
    } finally {
      setSaving(false);
    }
  }
  async function handleDelete(id) {
    const ok = await confirmDelete({
      text: 'ลบการจับคู่ค่ารหัสนี้?'
    });
    if (!ok) return;
    try {
      await deleteCodeAlias(id);
      toastSuccess('ลบแล้ว');
      await load();
    } catch (err) {
      toastError(err.message || 'ลบไม่สำเร็จ');
    }
  }
  async function handleUpload(e) {
    const file = e.target.files?.[0];
    e.target.value = '';
    await uploadFile(file);
  }
  async function uploadFile(file) {
    if (!file) return;
    setUploading(true);
    try {
      const res = await uploadCodeAliases(file, componentType);
      const parts = [`นำเข้าแล้ว ${res.imported ?? 0} รายการ`];
      if (res.updated) parts.push(`อัปเดต ${res.updated}`);
      if (res.skipped) parts.push(`ข้าม ${res.skipped}`);
      toastSuccess(parts.join(' · '));
      if (Array.isArray(res.problems) && res.problems.length > 0) {
        toastError(res.problems.slice(0, 5).join('\n'));
      }
      await load();
    } catch (err) {
      toastError(err.message || 'อัปโหลดไม่สำเร็จ');
    } finally {
      setUploading(false);
    }
  }
  const {
    dragging: aliasDragging,
    stateClass: aliasDropState,
    dropProps: aliasDropProps
  } = useFileDrop({
    accept: '.xlsx,.xls,.csv',
    disabled: uploading,
    onFile: uploadFile,
    onReject: (file, hint) => toastError(`ไฟล์ "${file.name}" ไม่รองรับ — ต้องเป็น ${hint}`)
  });

  function handleDownloadSample() {
    const columns = [{
      key: 'new',
      header: 'New (ค่าใหม่)',
      type: 'text'
    }, {
      key: 'old',
      header: 'Old (ค่าเดิม)',
      type: 'text'
    }, {
      key: 'kind',
      header: 'kind',
      type: 'text'
    }, {
      key: 'note',
      header: 'note',
      type: 'text'
    }];
    const rows = [{
      new: 'TNN-YN23993',
      old: 'YN23993',
      kind: 'machine',
      note: 'ตัวอย่าง Machine No. (ค่าเดิมต้องมีในระบบ)'
    }, {
      new: 'KQ-3000/NEW',
      old: 'KQ3000045093',
      kind: 'sn',
      note: 'ตัวอย่าง S/N (ค่าเดิมต้องมีในระบบ)'
    }, {
      new: 'YN22-E00849',
      old: 'YN22E00849FA',
      kind: 'pn',
      note: 'ตัวอย่าง P/N (ค่าเดิมต้องมีในระบบ)'
    }];
    const blob = buildStyledXlsxBlob({
      sheetName: 'Change Format Part',
      columns,
      rows
    });
    downloadBlob(blob, 'change-format-part-ตัวอย่าง.xlsx');
  }
  const body = <>
      <div className="fmt-form">
        <div className="fmt-field">
          <label className="fmt-label">ชนิดรหัส</label>
          <SelectField value={kind} onChange={setKind} options={[{
          value: 'machine',
          label: 'Machine No.'
        }, {
          value: 'sn',
          label: 'S/N'
        }, {
          value: 'pn',
          label: 'P/N'
        }]} />
        </div>
        <div className="fmt-field">
          <label className="fmt-label">New (ค่าใหม่) ({kindText})</label>
          <input className="fmt-input" value={fromCode} onChange={e => setFromCode(e.target.value)} placeholder="" />
        </div>
        <div className="fmt-field">
          <label className="fmt-label">Old (ค่าเดิม) ({kindText})</label>
          <input className="fmt-input" value={toSerial} onChange={e => setToSerial(e.target.value)} placeholder="" />
        </div>
        <div className="fmt-field">
          <label className="fmt-label">หมายเหตุ (ไม่บังคับ)</label>
          <input className="fmt-input" value={note} onChange={e => setNote(e.target.value)} />
        </div>
      </div>

      <div className={['fmt-actions', aliasDropState].filter(Boolean).join(' ')} {...aliasDropProps}>
        <button className="wh-issue-btn fmt-action-btn" type="button" onClick={handleDownloadSample}>
          ดาวน์โหลดตัวอย่าง
        </button>
        <label className="wh-issue-btn fmt-action-btn" style={{
        cursor: 'pointer'
      }} title="ลากไฟล์มาวางตรงนี้ก็ได้">
          <input type="file" accept=".xlsx,.xls,.csv" onChange={handleUpload} style={{
          display: 'none'
        }} disabled={uploading} />
          {uploading ? 'กำลังอัปโหลด...' : aliasDragging ? 'ปล่อยไฟล์ได้เลย' : 'อัปโหลดไฟล์'}
        </label>
        <button className="wh-issue-btn fmt-add-btn" onClick={handleAdd} disabled={saving}>
          {saving ? 'กำลังเพิ่ม...' : 'เพิ่ม'}
        </button>
      </div>

      <div className="fmt-table-wrap">
        <table className="wh-table" style={{
        width: '100%'
      }}>
          <thead>
            <tr>
              <th>ชนิด</th>
              <th>New (ค่าใหม่)</th>
              <th>→ Old (ค่าเดิม)</th>
              <th>หมายเหตุ</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && <tr>
                <td colSpan={5} className="wh-empty-cell">กำลังโหลด...</td>
              </tr>}
            {!loading && rows.length === 0 && <tr>
                <td colSpan={5} className="wh-empty-cell">ยังไม่มีการจับคู่ค่ารหัส</td>
              </tr>}
            {!loading && rows.map(r => <tr key={r.id}>
                  <td>{KIND_LABEL[r.kind] || '—'}</td>
                  <td style={codeStyle}>{r.new}</td>
                  <td style={codeStyle}>{r.to_serial_no}</td>
                  <td>{r.note || '—'}</td>
                  <td className="wh-cell-action">
                    <button className="qa-fail-btn" onClick={() => handleDelete(r.id)}>ลบ</button>
                  </td>
                </tr>)}
          </tbody>
        </table>
      </div>
    </>;
  if (embedded) return body;
  return <Collapsible title="Change Format Part" defaultOpen>
      {body}
    </Collapsible>;
}
export function PreviewResult({
  result
}) {
  if (!result) return null;
  if (result.headerFound === false) {
    return <div style={{
      marginTop: 10,
      padding: '10px 12px',
      background: '#fef2f2',
      borderRadius: 10,
      fontSize: 13,
      color: '#b42318'
    }}>
        {result.message || 'หาหัวตารางไม่เจอในไฟล์นี้'}
      </div>;
  }
  const matched = result.matched || [];
  const missing = result.missing || [];
  const extra = result.extra || [];
  const chip = (text, bg, color) => <span key={text} style={{
    background: bg,
    color,
    borderRadius: 999,
    padding: '2px 10px',
    fontSize: 12,
    ...codeStyle
  }}>
      {text}
    </span>;
  return <div style={{
    marginTop: 10,
    padding: '12px 14px',
    background: '#f8fafc',
    borderRadius: 10,
    fontSize: 13
  }}>
      <div style={{
      marginBottom: 6
    }}>
        พบหัวตารางแถวที่ {result.headerRow ?? '—'} — ไฟล์: <strong>{result.file}</strong>
      </div>
      <div style={{
      display: 'flex',
      flexWrap: 'wrap',
      gap: 6,
      alignItems: 'center',
      marginBottom: missing.length || extra.length ? 8 : 0
    }}>
        <span style={{
        color: '#16a34a',
        fontWeight: 600
      }}>แม็ปได้ {matched.length}:</span>
        {matched.length ? matched.map(m => chip(typeof m === 'string' ? m : m.label, '#dcfce7', '#166534')) : <span style={{
        color: '#94a3b8'
      }}>—</span>}
      </div>
      {missing.length > 0 && <div style={{
      display: 'flex',
      flexWrap: 'wrap',
      gap: 6,
      alignItems: 'center',
      marginBottom: extra.length ? 8 : 0
    }}>
          <span style={{
        color: '#b45309',
        fontWeight: 600
      }}>ไม่พบในไฟล์ {missing.length}:</span>
          {missing.map(m => chip(m, '#fef3c7', '#92400e'))}
        </div>}
      {extra.length > 0 && <div style={{
      display: 'flex',
      flexWrap: 'wrap',
      gap: 6,
      alignItems: 'center'
    }}>
          <span style={{
        color: '#2563eb',
        fontWeight: 600
      }}>คอลัมน์ใหม่ (จะถูกเก็บไว้) {extra.length}:</span>
          {extra.map(m => chip(m, '#dbeafe', '#1e40af'))}
        </div>}
    </div>;
}
export function ExtraColumnsCell({
  json,
  previewCount = 1
}) {
  const [expanded, setExpanded] = useState(false);
  let obj = null;
  try {
    obj = json ? JSON.parse(json) : null;
  } catch {
    obj = null;
  }
  const HIDDEN_EXTRA = new Set(['country', 'countryname', 'exportcountry', 'ประเทศ', 'ปลายทาง', 'ส่งออกไปประเทศ']);
  const normKey = k => String(k).replace(/^\[\+\]\s*/, '').toLowerCase().replace(/[\s_./-]/g, '');
  const entries = obj ? Object.entries(obj).filter(([k]) => !HIDDEN_EXTRA.has(normKey(k))) : [];
  if (entries.length === 0) return <span style={{
    color: '#cbd5e1'
  }}>—</span>;
  const limit = Math.max(1, previewCount);
  const hiddenCount = entries.length - limit;
  const collapsible = hiddenCount > 0;
  const visible = collapsible && !expanded ? entries.slice(0, limit) : entries;
  return <div style={{
    display: 'flex',
    flexDirection: 'column',
    gap: 5,
    minWidth: 160
  }}>
      {visible.map(([k, v]) => {
      const label = k.replace(/^\[\+\]\s*/, '');
      return <div key={k} title={`${label}: ${v}`} style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1,
        padding: '5px 9px',
        background: '#f8fafc',
        border: '1px solid #e5e9f0',
        borderLeft: '3px solid #60a5fa',
        borderRadius: 7
      }}>
            <span style={{
          fontSize: 10,
          fontWeight: 700,
          letterSpacing: 0.3,
          color: '#64748b',
          textTransform: 'uppercase',
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis'
        }}>
              {label}
            </span>
            <span style={{
          fontSize: 12.5,
          color: '#0f172a',
          fontWeight: 500,
          ...codeStyle
        }}>
              {String(v) || '—'}
            </span>
          </div>;
    })}
      {collapsible && <button type="button" onClick={e => {
      e.stopPropagation();
      setExpanded(v => !v);
    }} style={{
      alignSelf: 'flex-start',
      marginTop: 1,
      padding: '3px 9px',
      border: '1px solid #dbe3ee',
      borderRadius: 999,
      background: '#fff',
      color: '#2563eb',
      fontSize: 11.5,
      fontWeight: 600,
      cursor: 'pointer',
      lineHeight: 1.5
    }}>
          {expanded ? 'ดูน้อยลง' : `ดูเพิ่ม (+${hiddenCount})`}
        </button>}
    </div>;
}
export function ChangePreview({
  result
}) {
  if (!result) return null;
  if (result.headerFound === false) {
    return <div style={{
      marginTop: 10,
      padding: '10px 12px',
      background: '#fef2f2',
      borderRadius: 10,
      fontSize: 13,
      color: '#b42318'
    }}>
        {result.message || 'หาหัวตารางไม่เจอในไฟล์นี้'}
      </div>;
  }
  const s = result.summary || {};
  const rows = result.rows || [];
  const extra = result.extra || [];
  const missing = result.missing || [];
  const matched = result.matched || [];
  const keyLabel = result.keyLabel || 'Serial No.';
  const coreFields = result.coreFields && result.coreFields.length ? result.coreFields : ['P/N', 'S/N', 'IT Controller', 'IMEI'];
  const stat = (label, value, bg, color) => <div style={{
    background: bg,
    color,
    borderRadius: 10,
    padding: '8px 12px',
    minWidth: 92,
    textAlign: 'center'
  }}>
      <div style={{
      fontSize: 20,
      fontWeight: 700,
      lineHeight: 1
    }}>{value ?? 0}</div>
      <div style={{
      fontSize: 12,
      marginTop: 2
    }}>{label}</div>
    </div>;
  const badge = status => {
    const map = {
      NEW: ['#dcfce7', '#166534'],
      UPDATED: ['#dbeafe', '#1e40af'],
      CHANGED: ['#fef3c7', '#92400e']
    };
    const [bg, color] = map[status] || ['#f1f5f9', '#475569'];
    return <span style={{
      background: bg,
      color,
      borderRadius: 999,
      padding: '2px 9px',
      fontSize: 12,
      fontWeight: 600
    }}>{status}</span>;
  };
  const chip = (text, bg, color) => <span key={text} style={{
    background: bg,
    color,
    borderRadius: 999,
    padding: '2px 10px',
    fontSize: 12,
    ...codeStyle
  }}>
      {text}
    </span>;
  return <div style={{
    marginTop: 10,
    padding: '12px 14px',
    background: '#f8fafc',
    borderRadius: 10,
    fontSize: 13
  }}>
      <div style={{
      marginBottom: 8
    }}>
        ตรวจไฟล์ <strong>{result.file}</strong> (หัวตารางแถวที่ {result.headerRow ?? '—'}) — ยังไม่บันทึก กดอัปโหลดเพื่อยืนยัน
      </div>
      <div style={{
      display: 'flex',
      flexWrap: 'wrap',
      gap: 8,
      marginBottom: 10
    }}>
        {stat('ทั้งหมด', s.total, '#eef2ff', '#3730a3')}
        {stat('เพิ่มใหม่', s.new, '#dcfce7', '#166534')}
        {stat('อัปเดต', s.updated, '#dbeafe', '#1e40af')}
        {stat('ค่าเปลี่ยน', s.changed, '#fef3c7', '#92400e')}
        {stat('เหมือนเดิม', s.unchanged, '#f1f5f9', '#475569')}
      </div>

      {(matched.length > 0 || missing.length > 0) && <div style={{
      display: 'flex',
      flexWrap: 'wrap',
      gap: 6,
      alignItems: 'center',
      marginBottom: 8
    }}>
          <span style={{
        color: '#16a34a',
        fontWeight: 600
      }}>แม็ปได้ {matched.length}</span>
          {missing.length > 0 && <>
              <span style={{
          color: '#b45309',
          fontWeight: 600,
          marginLeft: 6
        }}>ไม่พบในไฟล์ {missing.length}:</span>
              {missing.map(m => chip(m, '#fef3c7', '#92400e'))}
            </>}
        </div>}

      {extra.length > 0 && <div style={{
      display: 'flex',
      flexWrap: 'wrap',
      gap: 6,
      alignItems: 'center',
      marginBottom: 8
    }}>
          <span style={{
        color: '#2563eb',
        fontWeight: 600
      }}>คอลัมน์ใหม่ (จะถูกเก็บไว้) {extra.length}:</span>
          {extra.map(m => chip(m, '#dbeafe', '#1e40af'))}
        </div>}

      {s.changed > 0 && <div style={{
      marginBottom: 8,
      color: '#92400e'
    }}>
          ⚠ มี {s.changed} แถวที่ค่าหลัก ({coreFields.join(' · ')}) เปลี่ยน — ตรวจก่อนยืนยัน
        </div>}

      {rows.length > 0 ? <div style={{
      maxHeight: 320,
      overflow: 'auto',
      border: '1px solid #e2e8f0',
      borderRadius: 8
    }}>
          <table className="wh-table" style={{
        width: '100%'
      }}>
            <thead>
              <tr>
                <th>สถานะ</th>
                <th>{keyLabel}</th>
                <th>ฟิลด์ที่เปลี่ยน (เดิม → ใหม่)</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r, i) => <tr key={`${r.serial ?? r.key}-${i}`}>
                  <td>{badge(r.status)}</td>
                  <td style={codeStyle}>{r.serial ?? r.key ?? '—'}</td>
                  <td>
                    {(r.diffs || []).length === 0 ? <span style={{
                color: '#94a3b8'
              }}>{r.status === 'NEW' ? 'แถวใหม่' : '—'}</span> : <div style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 2
              }}>
                        {r.diffs.map((d, j) => <div key={j} style={{
                  fontSize: 12
                }}>
                            <span style={{
                    color: '#64748b'
                  }}>{d.field}: </span>
                            <span style={{
                    ...codeStyle,
                    color: '#b91c1c'
                  }}>{d.old || '(ว่าง)'}</span>
                            <span style={{
                    color: '#94a3b8'
                  }}> → </span>
                            <span style={{
                    ...codeStyle,
                    color: '#15803d'
                  }}>{d.new || '(ว่าง)'}</span>
                          </div>)}
                      </div>}
                  </td>
                </tr>)}
            </tbody>
          </table>
        </div> : <div style={{
      color: '#64748b'
    }}>ทุกแถวเหมือนเดิม — อัปโหลดจะไม่เปลี่ยนแปลงข้อมูล</div>}

      {result.problems?.length > 0 && <ul style={{
      margin: '8px 0 0',
      paddingLeft: 18,
      color: '#b45309',
      fontSize: 12
    }}>
          {result.problems.map((p, i) => <li key={i}>{p}</li>)}
        </ul>}
    </div>;
}
export function MasterDataEditModal({
  row,
  componentOptions = [],
  itcLabel = 'IT Controller no.',
  onClose,
  onSaved
}) {
  const [form, setForm] = useState({
    Name: row.Name || '',
    Model: row.Model || '',
    ComponentType: row.ComponentType || '',
    PartNo: row.PartNo || '',
    SerialNo: row.SerialNo || '',
    ITControllerNo: row.ITControllerNo || '',
    IMEI: row.IMEI || ''
  });
  const [saving, setSaving] = useState(false);
  const set = k => e => setForm(f => ({
    ...f,
    [k]: e.target.value
  }));
  async function handleSave() {
    if (!form.SerialNo.trim() && !form.Name.trim()) {
      toastError('อย่างน้อยต้องมี Serial No. หรือ Part Name');
      return;
    }
    setSaving(true);
    try {
      const patch = {
        Name: form.Name.trim(),
        Model: form.Model.trim(),
        ComponentType: form.ComponentType.trim(),
        PartNo: form.PartNo.trim(),
        SerialNo: form.SerialNo.trim(),
        ITControllerNo: form.ITControllerNo.trim(),
        IMEI: form.IMEI.trim()
      };
      await saveWithGuard(patch);
      toastSuccess('บันทึกการแก้ไขแล้ว');
      onSaved && onSaved();
      onClose && onClose();
    } catch (err) {
      if (err?.message !== '__CANCELLED__') {
        toastError(err.message || 'บันทึกไม่สำเร็จ');
      }
    } finally {
      setSaving(false);
    }
  }
  async function saveWithGuard(patch) {
    try {
      await updateMasterData(row.ID, patch);
    } catch (err) {
      if (err?.status === 409 && err?.data?.blocked) {
        const refs = err.data.refs || {};
        const ok = await confirmDelete({
          title: 'ยืนยันการแก้ข้อมูลกุญแจ',
          text: (err.message || 'แถวนี้ถูกใช้ยืนยัน/จับคู่ไปแล้ว') + `\n\nรายการที่อ้างอิงอยู่: PartCheck ${refs.part_check || 0}, MFG ${refs.mfg_assembly || 0}, ` + `Matching ${refs.matching_assembly || 0}, Import License ${refs.import_license || 0}` + '\n\nยืนยันแก้ต่อ (อาจกระทบการ match เดิม)?',
          confirmText: 'ยืนยันแก้ (force)'
        });
        if (!ok) throw new Error('__CANCELLED__');
        await updateMasterData(row.ID, patch, {
          force: true
        });
        return;
      }
      throw err;
    }
  }
  const field = (label, key, mono = false) => <div className="fmt-field">
      <label className="fmt-label">{label}</label>
      <input className={'fmt-input' + (mono ? ' fmt-input-mono' : '')} value={form[key]} onChange={set(key)} />
    </div>;
  return <div className="wh-modal-overlay" onClick={onClose}>
      <div className="wh-modal" style={{
      maxWidth: 560
    }} onClick={e => e.stopPropagation()}>
        <h3 className="wh-modal-title">แก้ไขทะเบียน Master Data</h3>

        <div className="fmt-form fmt-form-compact" style={{
        marginTop: 12
      }}>
          {field('Part Name', 'Name')}
          {field('Model', 'Model')}
          <div className="fmt-field">
            <label className="fmt-label">ชนิดอะไหล่</label>
            {componentOptions.length > 0 ? <SelectField value={form.ComponentType} onChange={v => setForm(f => ({
            ...f,
            ComponentType: v
          }))} options={componentOptions} /> : <input className="fmt-input" value={form.ComponentType} onChange={set('ComponentType')} />}
          </div>
          {form.ComponentType === 'it_controller' && field('Part No.', 'PartNo', true)}
          {field('Serial No.', 'SerialNo', true)}
          {field(itcLabel, 'ITControllerNo', true)}
          {form.ComponentType === 'it_controller' && field('IMEI', 'IMEI', true)}
        </div>

        <p style={{
        fontSize: 12,
        color: '#94a3b8',
        marginTop: 10
      }}>
          แก้เพื่อให้ตรงกับ format ใหม่ที่หน้างานใช้ — เว้นว่างได้ในช่องที่อะไหล่ชนิดนั้นไม่มี
        </p>

        <div className="wh-modal-actions">
          <button className="wh-modal-cancel" onClick={onClose} disabled={saving}>ยกเลิก</button>
          <button className="wh-issue-btn" onClick={handleSave} disabled={saving}>
            {saving ? 'กำลังบันทึก...' : 'บันทึก'}
          </button>
        </div>
      </div>
    </div>;
}