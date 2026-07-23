import { useEffect, useMemo, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import FileDropZone from '../components/Filedropzone.jsx'
import SelectField from '../components/Selectfield.jsx'
import {
  getDocuments,
  uploadDocument,
  getImportLicenses,
  saveImportLicense,
  uploadSerialList,
  getUnits,
  receiveUnit,
  allocateUnits,
  allocateSplit,
  exportUnit,
  getExportLicenses,
  createExportLicense,
  downloadExportAttachment,
  getAlerts,
  traceUnit,
} from '../api/itController.js'

const navItems = [
  { to: '/warehouse', label: 'จ่ายของ (FIFO & S/O)', icon: '📦' },
  { to: '/warehouse/confirm', label: 'Part Confirmation', icon: '✅' },
  { to: '/warehouse/it-controller', label: 'IT Controller (กสทช.)', icon: '📡' },
]

const TABS = [
  { key: 'docs', label: '1. เอกสาร & ใบอนุญาตนำเข้า' },
  { key: 'units', label: '2. ทะเบียนเครื่อง' },
  { key: 'allocate', label: '3. จัดสรรประเทศ' },
  { key: 'export', label: '4. ใบอนุญาตนำออก' },
  { key: 'trace', label: '5. ตรวจสอบย้อนกลับ' },
]

const STATUS_LABEL = {
  IMPORTED: 'นำเข้าทะเบียน',
  RECEIVED: 'รับเข้าคลัง',
  ALLOCATED: 'จัดสรรประเทศแล้ว',
  LICENSED: 'มีใบนำออก',
  EXPORTED: 'ส่งออกแล้ว',
}

const today = () => new Date().toISOString().slice(0, 10)
const fmtDate = (d) => (d && !d.startsWith('0001') ? new Date(d).toLocaleDateString('th-TH') : '—')

export default function ITControllerPage() {
  const [tab, setTab] = useState('docs')
  const [toast, setToast] = useState('')
  const [error, setError] = useState('')

  const [alerts, setAlerts] = useState([])
  const [documents, setDocuments] = useState([])
  const [importLicenses, setImportLicenses] = useState([])
  const [exportLicenses, setExportLicenses] = useState([])
  const [units, setUnits] = useState([])

  function flash(message) {
    setToast(message)
    setTimeout(() => setToast(''), 4000)
  }

  async function reload() {
    setError('')
    try {
      const [a, d, il, el, u] = await Promise.all([
        getAlerts(),
        getDocuments(),
        getImportLicenses(),
        getExportLicenses(),
        getUnits(),
      ])
      setAlerts(a || [])
      setDocuments(d || [])
      setImportLicenses(il || [])
      setExportLicenses(el || [])
      setUnits(u || [])
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => {
    reload()
  }, [])

  const counts = useMemo(() => {
    const base = { total: units.length, IMPORTED: 0, RECEIVED: 0, ALLOCATED: 0, LICENSED: 0, EXPORTED: 0 }
    for (const u of units) base[u.Status] = (base[u.Status] || 0) + 1
    return base
  }, [units])

  return (
    <AppShell navItems={navItems} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">IT Controller — ทะเบียน กสทช.</h2>
          <p className="wh-subtitle">ทุกอย่างอ้างอิงด้วย IT Controller No. 12 หลัก</p>
        </div>
      </div>

      {toast && <div className="wh-toast">{toast}</div>}
      {error && (
        <p className="form-error" role="alert">
          {error}
        </p>
      )}

      <AlertBanner alerts={alerts} />

      <div className="dash-stats-row wh-stats-row">
        <StatCard label="เครื่องทั้งหมด" value={counts.total} icon="▦" tone="blue" />
        <StatCard label="รับเข้าคลังแล้ว" value={counts.RECEIVED} icon="📥" tone="yellow" />
        <StatCard label="รอใบนำออก" value={counts.ALLOCATED} icon="⏳" tone="yellow" />
        <StatCard label="ส่งออกแล้ว" value={counts.EXPORTED} icon="✈" tone="green" />
      </div>

      <div className="wh-toolbar-row itc-tabs-scroll">
        <div className="vr-tabs">
          {TABS.map((t) => (
            <button
              key={t.key}
              className={'vr-tab' + (tab === t.key ? ' vr-tab-active' : '')}
              onClick={() => setTab(t.key)}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {tab === 'docs' && (
        <DocumentsTab
          documents={documents}
          importLicenses={importLicenses}
          onDone={(msg) => {
            flash(msg)
            reload()
          }}
          onError={setError}
        />
      )}

      {tab === 'units' && (
        <UnitsTab
          units={units}
          onDone={(msg) => {
            flash(msg)
            reload()
          }}
          onError={setError}
        />
      )}

      {tab === 'allocate' && (
        <AllocateTab
          units={units}
          importLicenses={importLicenses}
          onDone={(msg) => {
            flash(msg)
            reload()
          }}
          onError={setError}
        />
      )}

      {tab === 'export' && (
        <ExportTab
          units={units}
          importLicenses={importLicenses}
          exportLicenses={exportLicenses}
          onDone={(msg) => {
            flash(msg)
            reload()
          }}
          onError={setError}
        />
      )}

      {tab === 'trace' && <TraceTab onError={setError} />}
    </AppShell>
  )
}

// ─────────────────────────────────────────────────────────────────────────────

function StatCard({ label, value, icon, tone }) {
  return (
    <div className="dash-stat-card">
      <div className="dash-stat-label">
        <span>{label}</span>
        <span className={`dash-stat-icon dash-icon-${tone}`}>{icon}</span>
      </div>
      <div className="dash-stat-value">{value}</div>
    </div>
  )
}

function AlertBanner({ alerts }) {
  if (!alerts.length) return null

  return (
    <div className="itc-alert-list">
      {alerts.map((a, i) => (
        <div key={i} className={`itc-alert itc-alert-${a.level.toLowerCase()}`}>
          <span className="itc-alert-tag">{a.level}</span>
          <span>{a.message}</span>
        </div>
      ))}
    </div>
  )
}

// ── Tab 1: เอกสาร PDF + หัวใบอนุญาตนำเข้า + Serial List ──────────────────────

function DocumentsTab({ documents, importLicenses, onDone, onError }) {
  const [doc, setDoc] = useState({ docType: 'INVOICE', docNo: '', invoiceNo: '', poNo: '', file: null })
  const [lic, setLic] = useState({
    license_no: '',
    invoice_no: '',
    po_no: '',
    declaration_no: '',
    brand: 'JRC MOBILITY',
    model: 'JRN-260K',
    part_no: '',
    qty: '',
    issue_date: today(),
  })
  const [serial, setSerial] = useState({ file: null, importLicenseNo: '' })
  const [result, setResult] = useState(null)
  const [busy, setBusy] = useState('')

  const selectedLicense = importLicenses.find((l) => l.LicenseNo === serial.importLicenseNo)

  async function submitDoc() {
    if (!doc.file || !doc.docNo) return onError('กรุณาเลือกไฟล์ PDF และกรอกเลขที่เอกสาร')
    setBusy('doc')
    try {
      await uploadDocument(doc)
      setDoc({ docType: doc.docType, docNo: '', invoiceNo: '', poNo: '', file: null })
      onDone('บันทึกเอกสารแล้ว')
    } catch (e) {
      onError(e.message)
    } finally {
      setBusy('')
    }
  }

  async function submitLicense() {
    setBusy('lic')
    try {
      await saveImportLicense({ ...lic, qty: Number(lic.qty) })
      onDone(`บันทึกใบอนุญาตนำเข้า ${lic.license_no} แล้ว`)
    } catch (e) {
      onError(e.message)
    } finally {
      setBusy('')
    }
  }

  async function submitSerial() {
    if (!serial.file || !selectedLicense) return onError('กรุณาเลือกใบอนุญาตนำเข้าและไฟล์ Serial List')
    setBusy('serial')
    setResult(null)
    try {
      const res = await uploadSerialList({
        file: serial.file,
        invoiceNo: selectedLicense.InvoiceNo,
        poNo: selectedLicense.PONo,
        importLicenseNo: selectedLicense.LicenseNo,
      })
      setResult(res)
      onDone(`นำเข้าทะเบียนแล้ว ${res.created} เครื่องใหม่ / อัปเดต ${res.updated}`)
    } catch (e) {
      onError(e.message)
    } finally {
      setBusy('')
    }
  }

  return (
    <>
      <section className="itc-card">
        <h3 className="tsf-form-title">อัปโหลดเอกสาร PDF</h3>
        <p className="wh-subtitle">
          Invoice / PO / ใบอนุญาต เข้ามาเป็น PDF ระบบเก็บไฟล์ไว้เป็นหลักฐาน ส่วนเลขที่ให้คีย์กำกับตรงนี้
        </p>

        <div className="itc-form-grid">
          <label className="itc-field">
            <span>ประเภทเอกสาร</span>
            <SelectField
              value={doc.docType}
              onChange={(docType) => setDoc({ ...doc, docType })}
              options={[
                { value: 'INVOICE', label: 'Invoice' },
                { value: 'PO', label: 'Purchase Order' },
                { value: 'IMPORT_LICENSE', label: 'ใบอนุญาตนำเข้า' },
                { value: 'EXPORT_LICENSE', label: 'ใบอนุญาตนำออก' },
                { value: 'SERIAL_LIST', label: 'Serial List' },
              ]}
            />
          </label>

          <label className="itc-field">
            <span>เลขที่บนเอกสาร</span>
            <input value={doc.docNo} onChange={(e) => setDoc({ ...doc, docNo: e.target.value })} placeholder="TQ60610" />
          </label>

          <label className="itc-field">
            <span>Invoice No. (ถ้ามี)</span>
            <input value={doc.invoiceNo} onChange={(e) => setDoc({ ...doc, invoiceNo: e.target.value })} />
          </label>

          <label className="itc-field">
            <span>P.O. No. (ถ้ามี)</span>
            <input value={doc.poNo} onChange={(e) => setDoc({ ...doc, poNo: e.target.value })} />
          </label>

        </div>

        <div className="fdz-row">
          <FileDropZone
            file={doc.file}
            onSelect={(file) => setDoc({ ...doc, file })}
            accept=".pdf"
            label="ไฟล์เอกสาร PDF"
            hint="ลากไฟล์มาวาง หรือกดเพื่อเลือก (.pdf)"
            disabled={busy === 'doc'}
          />
          <button className="wh-issue-btn" onClick={submitDoc} disabled={busy === 'doc' || !doc.file}>
            {busy === 'doc' ? 'กำลังบันทึก...' : 'บันทึกเอกสาร'}
          </button>
        </div>
      </section>

      <section className="itc-card">
        <h3 className="tsf-form-title">บันทึกหัวใบอนุญาตนำเข้า</h3>
        <p className="wh-subtitle">วันหมดอายุคำนวณให้อัตโนมัติ = วันที่ออก + 6 เดือน</p>

        <div className="itc-form-grid">
          {[
            ['license_no', 'เลขใบอนุญาตนำเข้า', 'E05036901604'],
            ['invoice_no', 'Invoice No.', 'TQ60610'],
            ['po_no', 'P.O. No.', '6910187190'],
            ['declaration_no', 'เลขใบขนสินค้าขาเข้า', 'A0220690606031'],
            ['part_no', 'Part No.', 'YN22E00849FA'],
            ['qty', 'จำนวนบนใบอนุญาต', '35'],
          ].map(([key, label, ph]) => (
            <label className="itc-field" key={key}>
              <span>{label}</span>
              <input value={lic[key]} placeholder={ph} onChange={(e) => setLic({ ...lic, [key]: e.target.value })} />
            </label>
          ))}

          <label className="itc-field">
            <span>วันที่ออกใบอนุญาต</span>
            <input type="date" value={lic.issue_date} onChange={(e) => setLic({ ...lic, issue_date: e.target.value })} />
          </label>
        </div>

        <button className="wh-issue-btn" onClick={submitLicense} disabled={busy === 'lic'}>
          {busy === 'lic' ? 'กำลังบันทึก...' : 'บันทึกใบอนุญาต'}
        </button>
      </section>

      <section className="itc-card">
        <h3 className="tsf-form-title">นำเข้า Serial List (Excel) → สร้างทะเบียนเครื่อง</h3>
        <p className="wh-subtitle">
          รองรับทั้งไฟล์ SERIAL NO. (หัวอังกฤษ) และบัญชีแสดงหมายเลขเครื่อง (หัวไทย) ระบบจะเทียบจำนวนกับใบอนุญาตให้
        </p>

        <div className="itc-form-grid">
          <label className="itc-field">
            <span>ใบอนุญาตนำเข้าที่จะผูก</span>
            <SelectField
              value={serial.importLicenseNo}
              onChange={(importLicenseNo) => setSerial({ ...serial, importLicenseNo })}
              placeholder="— เลือกใบอนุญาต —"
              options={importLicenses.map((l) => ({
                value: l.LicenseNo,
                label: `${l.LicenseNo} · ${l.InvoiceNo} · PO ${l.PONo} · ${l.Qty} เครื่อง`,
              }))}
            />
          </label>

        </div>

        <div className="fdz-row">
          <FileDropZone
            file={serial.file}
            onSelect={(file) => setSerial({ ...serial, file })}
            accept=".xlsx,.xls"
            label="ไฟล์ Serial List"
            hint="ลากไฟล์ Excel มาวาง หรือกดเพื่อเลือก (.xlsx, .xls)"
            disabled={busy === 'serial'}
          />
          <button
            className="wh-issue-btn"
            onClick={submitSerial}
            disabled={busy === 'serial' || !serial.file || !serial.importLicenseNo}
          >
            {busy === 'serial' ? 'กำลังนำเข้า...' : 'นำเข้าทะเบียน'}
          </button>
        </div>

        {result && (
          <div className="itc-result">
            <p>
              อ่านได้ {result.total} แถว · สร้างใหม่ {result.created} · อัปเดต {result.updated} · ข้าม {result.skipped}
            </p>
            {result.warnings?.map((w, i) => (
              <p key={i} className="itc-warn">
                ⚠ {w}
              </p>
            ))}
          </div>
        )}
      </section>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ใบอนุญาตนำเข้า</th>
              <th>Invoice</th>
              <th>P.O.</th>
              <th>จำนวนบนใบ</th>
              <th>ในระบบ</th>
              <th>ส่งออกแล้ว</th>
              <th>หมดอายุ</th>
              <th>เหลือ (วัน)</th>
            </tr>
          </thead>
          <tbody>
            {importLicenses.map((l) => (
              <tr key={l.LicenseNo}>
                <td data-label="ใบอนุญาตนำเข้า" className="wh-cell-head">{l.LicenseNo}</td>
                <td data-label="Invoice">{l.InvoiceNo}</td>
                <td data-label="P.O.">{l.PONo}</td>
                <td data-label="จำนวนบนใบ">{l.Qty}</td>
                <td data-label="ในระบบ" className={l.unit_count !== l.Qty ? 'itc-cell-bad' : ''}>{l.unit_count}</td>
                <td data-label="ส่งออกแล้ว">{l.exported_count}</td>
                <td data-label="หมดอายุ">{fmtDate(l.ExpireDate)}</td>
                <td data-label="เหลือ (วัน)">{l.days_left}</td>
              </tr>
            ))}
            {!importLicenses.length && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
                  ยังไม่มีใบอนุญาตนำเข้าในระบบ
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ประเภท</th>
              <th>เลขที่</th>
              <th>ไฟล์</th>
              <th>อัปโหลดเมื่อ</th>
              <th>โดย</th>
            </tr>
          </thead>
          <tbody>
            {documents.map((d) => (
              <tr key={d.ID}>
                <td data-label="ประเภท" className="wh-cell-head">{d.DocType}</td>
                <td data-label="เลขที่">{d.DocNo}</td>
                <td data-label="ไฟล์">
                  <a href={d.FileURL} target="_blank" rel="noreferrer">
                    {d.FileName}
                  </a>
                </td>
                <td data-label="อัปโหลดเมื่อ">{fmtDate(d.UploadDate)}</td>
                <td data-label="โดย">{d.Name}</td>
              </tr>
            ))}
            {!documents.length && (
              <tr>
                <td colSpan={5} className="wh-empty-cell">
                  ยังไม่มีเอกสาร
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </>
  )
}

// ── Tab 2: ทะเบียนเครื่อง + สแกนรับเข้าคลัง ──────────────────────────────────

function UnitsTab({ units, onDone, onError }) {
  const [scan, setScan] = useState('')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')

  async function handleReceive(e) {
    e.preventDefault()
    const value = scan.trim()
    if (!value) return
    try {
      const unit = await receiveUnit(value)
      setScan('')
      onDone(`รับเข้าคลัง ${unit.ITControllerNo} แล้ว`)
    } catch (err) {
      onError(err.message)
    }
  }

  const rows = useMemo(() => {
    const term = search.trim().toLowerCase()
    return units.filter((u) => {
      if (statusFilter && u.Status !== statusFilter) return false
      if (!term) return true
      return [u.ITControllerNo, u.IMEI, u.SerialNo, u.Country, u.MachineNo]
        .filter(Boolean)
        .some((v) => String(v).toLowerCase().includes(term))
    })
  }, [units, search, statusFilter])

  return (
    <>
      <section className="itc-card">
        <h3 className="tsf-form-title">สแกนรับเข้าคลัง</h3>
        <p className="wh-subtitle">
          1 เครื่องมี 2 ป้าย — ป้ายบน S/N + IMEI, ป้ายล่าง P/N + IT Controller No. ยิงใบไหนก็ได้ ระบบหาเครื่องเจอเอง
        </p>
        <form className="itc-scan-row" onSubmit={handleReceive}>
          <input
            className="wh-search"
            autoFocus
            value={scan}
            onChange={(e) => setScan(e.target.value)}
            placeholder="ยิงบาร์โค้ดป้ายไหนก็ได้ — IT Controller No. / IMEI / S/N"
          />
          <button className="wh-issue-btn" type="submit">
            รับเข้าคลัง
          </button>
        </form>
      </section>

      <div className="wh-toolbar-row">
        <SelectField
          value={statusFilter}
          onChange={setStatusFilter}
          placeholder="ทุกสถานะ"
          options={[
            { value: '', label: 'ทุกสถานะ' },
            ...Object.entries(STATUS_LABEL).map(([k, v]) => ({ value: k, label: v })),
          ]}
        />
        <input
          className="wh-search"
          placeholder="ค้นด้วย IT Controller No. / IMEI / Serial"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>IT Controller No.</th>
              <th>IMEI</th>
              <th>Serial No.</th>
              <th>Invoice</th>
              <th>P.O.</th>
              <th>ใบนำเข้า</th>
              <th>ประเทศ</th>
              <th>ใบนำออก</th>
              <th>สถานะ</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((u) => (
              <tr key={u.ITControllerNo}>
                <td data-label="IT Controller No." className="wh-cell-head">{u.ITControllerNo}</td>
                <td data-label="IMEI">{u.IMEI}</td>
                <td data-label="Serial No.">{u.SerialNo}</td>
                <td data-label="Invoice">{u.InvoiceNo}</td>
                <td data-label="P.O.">{u.PONo}</td>
                <td data-label="ใบนำเข้า">{u.ImportLicenseNo}</td>
                <td data-label="ประเทศ">{u.Country || '—'}</td>
                <td data-label="ใบนำออก">{u.ExportLicenseNo || '—'}</td>
                <td data-label="สถานะ">
                  <span className={`itc-status itc-status-${u.Status.toLowerCase()}`}>
                    {STATUS_LABEL[u.Status] || u.Status}
                  </span>
                </td>
              </tr>
            ))}
            {!rows.length && (
              <tr>
                <td colSpan={9} className="wh-empty-cell">
                  ไม่พบข้อมูล
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </>
  )
}

// ── Tab 3: จัดสรรประเทศปลายทาง ───────────────────────────────────────────────

function AllocateTab({ units, importLicenses, onDone, onError }) {
  const [mode, setMode] = useState('split')

  // จัดสรรได้เฉพาะของที่รับเข้าคลังแล้ว หรือที่จัดสรรไว้แล้วแต่ยังไม่ได้ใบนำออก
  const pool = useMemo(
    () => units.filter((u) => u.Status === 'RECEIVED' || u.Status === 'ALLOCATED'),
    [units],
  )

  const unassigned = useMemo(() => units.filter((u) => u.Status === 'RECEIVED'), [units])

  return (
    <>
      <div className="wh-toolbar-row itc-tabs-scroll">
        <div className="vr-tabs">
          <button
            className={'vr-tab' + (mode === 'split' ? ' vr-tab-active' : '')}
            onClick={() => setMode('split')}
          >
            แบ่งเป็นชุด (หลายประเทศพร้อมกัน)
          </button>
          <button
            className={'vr-tab' + (mode === 'manual' ? ' vr-tab-active' : '')}
            onClick={() => setMode('manual')}
          >
            เลือกทีละเครื่อง
          </button>
        </div>
      </div>

      {mode === 'split' ? (
        <SplitPlan
          available={unassigned.length}
          importLicenses={importLicenses}
          onDone={onDone}
          onError={onError}
        />
      ) : (
        <ManualPick pool={pool} onDone={onDone} onError={onError} />
      )}

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ประเทศ</th>
              <th>จัดสรรแล้ว</th>
              <th>มีใบนำออก</th>
              <th>ส่งออกแล้ว</th>
            </tr>
          </thead>
          <tbody>
            {summarizeByCountry(units).map((row) => (
              <tr key={row.country}>
                <td data-label="ประเทศ" className="wh-cell-head">{row.country}</td>
                <td data-label="จัดสรรแล้ว">{row.allocated}</td>
                <td data-label="มีใบนำออก">{row.licensed}</td>
                <td data-label="ส่งออกแล้ว">{row.exported}</td>
              </tr>
            ))}
            {!summarizeByCountry(units).length && (
              <tr>
                <td colSpan={4} className="wh-empty-cell">
                  ยังไม่ได้จัดสรรประเทศให้เครื่องไหน
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </>
  )
}

function summarizeByCountry(units) {
  const map = new Map()

  for (const u of units) {
    if (!u.Country) continue

    const row = map.get(u.Country) || { country: u.Country, allocated: 0, licensed: 0, exported: 0 }

    if (u.Status === 'ALLOCATED') row.allocated++
    if (u.Status === 'LICENSED') row.licensed++
    if (u.Status === 'EXPORTED') row.exported++

    map.set(u.Country, row)
  }

  return [...map.values()].sort((a, b) => a.country.localeCompare(b.country))
}

// แผนแบ่งของ: กรอกประเทศ + จำนวนได้หลายบรรทัด ระบบตรวจยอดรวมก่อนบันทึกครั้งเดียว
function SplitPlan({ available, importLicenses, onDone, onError }) {
  const [rows, setRows] = useState([
    { country: '', qty: '' },
    { country: '', qty: '' },
  ])
  const [licenseNo, setLicenseNo] = useState('')
  const [busy, setBusy] = useState(false)

  const planned = rows.reduce((sum, r) => sum + (Number(r.qty) || 0), 0)
  const left = available - planned
  const duplicated = hasDuplicateCountry(rows)

  function update(i, key, value) {
    setRows(rows.map((r, idx) => (idx === i ? { ...r, [key]: value } : r)))
  }

  function addRow() {
    setRows([...rows, { country: '', qty: '' }])
  }

  function removeRow(i) {
    setRows(rows.length > 1 ? rows.filter((_, idx) => idx !== i) : rows)
  }

  function fillRest(i) {
    const others = rows.reduce((sum, r, idx) => (idx === i ? sum : sum + (Number(r.qty) || 0)), 0)
    update(i, 'qty', String(Math.max(available - others, 0)))
  }

  async function submit() {
    const splits = rows
      .filter((r) => r.country.trim() && Number(r.qty) > 0)
      .map((r) => ({ country: r.country.trim(), qty: Number(r.qty) }))

    if (!splits.length) return onError('กรุณากรอกประเทศและจำนวนอย่างน้อย 1 บรรทัด')
    if (planned > available) return onError(`ขอ ${planned} เครื่อง แต่มีของพร้อมจัดสรรแค่ ${available} เครื่อง`)

    setBusy(true)
    try {
      const res = await allocateSplit(splits, { importLicenseNo: licenseNo })
      setRows([{ country: '', qty: '' }])
      onDone(`แบ่งแล้ว ${res.total} เครื่อง เหลือยังไม่จัดสรร ${res.remaining} เครื่อง`)
    } catch (e) {
      onError(e.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="itc-card">
      <h3 className="tsf-form-title">แผนแบ่งของ</h3>
      <p className="wh-subtitle">
        กรอกให้ครบทุกประเทศก่อน แล้วกดบันทึกครั้งเดียว ระบบหยิบเครื่องเรียงตาม IT Controller No. ให้เอง
      </p>

      <label className="itc-field" style={{ maxWidth: 420, marginTop: 14 }}>
        <span>จำกัดเฉพาะใบอนุญาตนำเข้า (ไม่เลือก = ทุกใบ)</span>
        <SelectField
          value={licenseNo}
          onChange={setLicenseNo}
          placeholder="— ทุกใบ —"
          options={[
            { value: '', label: '— ทุกใบ —' },
            ...importLicenses.map((l) => ({
              value: l.LicenseNo,
              label: `${l.LicenseNo} · ${l.InvoiceNo}`,
            })),
          ]}
        />
      </label>

      {available === 0 && (
        <p className="itc-empty-hint">
          ยังไม่มีเครื่องที่พร้อมจัดสรร — ต้องนำเข้า Serial List ในแท็บ 1 แล้วสแกนรับเข้าคลังในแท็บ 2 ก่อน
        </p>
      )}

      <div className="itc-split-rows">
        {rows.map((row, i) => (
          <div className="itc-split-row" key={i}>
            <input
              list="itc-country-list"
              placeholder="ประเทศปลายทาง"
              value={row.country}
              onChange={(e) => update(i, 'country', e.target.value)}
            />
            <input
              type="number"
              min="0"
              placeholder="จำนวน"
              value={row.qty}
              onChange={(e) => update(i, 'qty', e.target.value)}
            />
            <button type="button" className="itc-ghost-btn" onClick={() => fillRest(i)}>
              เท่าที่เหลือ
            </button>
            <button
              type="button"
              className="itc-ghost-btn itc-ghost-danger"
              onClick={() => removeRow(i)}
              aria-label="ลบบรรทัด"
            >
              ✕
            </button>
          </div>
        ))}
      </div>

      <datalist id="itc-country-list">
        {COUNTRY_SUGGESTIONS.map((c) => (
          <option key={c} value={c} />
        ))}
      </datalist>

      <button type="button" className="itc-ghost-btn" onClick={addRow}>
        + เพิ่มประเทศ
      </button>

      <div className={'itc-split-total' + (planned > available || duplicated ? ' itc-split-total-bad' : '')}>
        <span>ของพร้อมจัดสรร {available} เครื่อง</span>
        <span>แบ่งไปแล้วในแผน {planned}</span>
        <span>{left >= 0 ? `เหลือ ${left}` : `เกินมา ${-left}`}</span>
      </div>

      {duplicated && <p className="itc-warn">⚠ มีประเทศซ้ำกันในแผน ระบบจะรวมยอดให้เป็นก้อนเดียว</p>}

      <button
        className="wh-issue-btn"
        onClick={submit}
        disabled={busy || planned === 0 || planned > available}
      >
        {busy ? 'กำลังบันทึก...' : `บันทึกการแบ่ง ${planned} เครื่อง`}
      </button>
    </section>
  )
}

function hasDuplicateCountry(rows) {
  const seen = new Set()

  for (const r of rows) {
    const key = r.country.trim().toLowerCase()
    if (!key) continue
    if (seen.has(key)) return true
    seen.add(key)
  }

  return false
}

// รายการแนะนำเฉย ๆ — ช่องประเทศเป็น <input list> ไม่ใช่ <select>
// พิมพ์ประเทศที่ไม่มีในลิสต์ได้เสมอ ระบบจะจัดรูปตัวพิมพ์ให้เอง
const COUNTRY_SUGGESTIONS = [
  'Indonesia',
  'Malaysia',
  'Japan',
  'Vietnam',
  'Philippines',
  'Singapore',
  'Thailand',
  'Myanmar',
  'Cambodia',
  'Laos',
  'India',
  'China',
  'Taiwan',
  'South Korea',
  'Australia',
  'New Zealand',
]

// โหมดเดิม — ใช้ตอนแก้รายตัวหรือย้ายประเทศทีหลัง
function ManualPick({ pool, onDone, onError }) {
  const [country, setCountry] = useState('')
  const [selected, setSelected] = useState([])

  function toggle(no) {
    setSelected((prev) => (prev.includes(no) ? prev.filter((x) => x !== no) : [...prev, no]))
  }

  async function submit() {
    if (!country.trim()) return onError('กรุณาระบุประเทศ')
    if (!selected.length) return onError('ยังไม่ได้เลือกเครื่อง')

    try {
      const res = await allocateUnits(selected, country.trim())
      setSelected([])
      onDone(`จัดสรรไป ${country} แล้ว ${res.allocated.length} เครื่อง`)
      if (res.rejected?.length) onError(res.rejected.join(' / '))
    } catch (e) {
      onError(e.message)
    }
  }

  return (
    <>
      <section className="itc-card">
        <div className="itc-scan-row">
          <input
            className="wh-search"
            list="itc-country-list"
            value={country}
            onChange={(e) => setCountry(e.target.value)}
            placeholder="ประเทศปลายทาง"
          />
          <button className="wh-issue-btn" onClick={submit} disabled={!selected.length}>
            ย้าย {selected.length} เครื่องไปประเทศนี้
          </button>
        </div>
        <datalist id="itc-country-list">
          {COUNTRY_SUGGESTIONS.map((c) => (
            <option key={c} value={c} />
          ))}
        </datalist>
      </section>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th></th>
              <th>IT Controller No.</th>
              <th>Serial No.</th>
              <th>ใบนำเข้า</th>
              <th>ประเทศปัจจุบัน</th>
              <th>สถานะ</th>
            </tr>
          </thead>
          <tbody>
            {pool.map((u) => (
              <tr key={u.ITControllerNo}>
                <td data-label="เลือก">
                  <input
                    type="checkbox"
                    checked={selected.includes(u.ITControllerNo)}
                    onChange={() => toggle(u.ITControllerNo)}
                  />
                </td>
                <td data-label="IT Controller No." className="wh-cell-head">{u.ITControllerNo}</td>
                <td data-label="Serial No.">{u.SerialNo}</td>
                <td data-label="ใบนำเข้า">{u.ImportLicenseNo}</td>
                <td data-label="ประเทศปัจจุบัน">{u.Country || '—'}</td>
                <td data-label="สถานะ">{STATUS_LABEL[u.Status]}</td>
              </tr>
            ))}
            {!pool.length && (
              <tr>
                <td colSpan={6} className="wh-empty-cell">
                  ยังไม่มีเครื่องที่รับเข้าคลัง
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </>
  )
}

// ── Tab 4: ใบอนุญาตนำออก + สแกนส่งออก ────────────────────────────────────────

function ExportTab({ units, importLicenses, exportLicenses, onDone, onError }) {
  const [form, setForm] = useState({ license_no: '', country: '', issue_date: today(), import_license_no: '' })
  const [scan, setScan] = useState('')
  const [scanCountry, setScanCountry] = useState('')

  const candidates = useMemo(
    () => units.filter((u) => u.Status === 'ALLOCATED' && (!form.country || u.Country === form.country)),
    [units, form.country],
  )

  const countries = useMemo(
    () => [...new Set(units.filter((u) => u.Country).map((u) => u.Country))],
    [units],
  )

  async function submit() {
    if (!form.license_no || !form.country) return onError('กรุณากรอกเลขใบอนุญาตและประเทศ')
    if (!candidates.length) return onError('ไม่มีเครื่องที่จัดสรรไว้สำหรับประเทศนี้')
    try {
      const res = await createExportLicense({
        ...form,
        it_controller_nos: candidates.map((u) => u.ITControllerNo),
      })
      onDone(`ผูกใบนำออก ${form.license_no} กับ ${res.attached.length} เครื่องแล้ว`)
      if (res.rejected?.length) onError(res.rejected.join(' / '))
    } catch (e) {
      onError(e.message)
    }
  }

  async function handleExport(e) {
    e.preventDefault()
    const value = scan.trim()
    if (!value) return
    try {
      const unit = await exportUnit(value, scanCountry)
      setScan('')
      onDone(`ส่งออก ${unit.ITControllerNo} → ${unit.Country} แล้ว`)
    } catch (err) {
      onError(err.message)
    }
  }

  return (
    <>
      <section className="itc-card">
        <h3 className="tsf-form-title">บันทึกใบอนุญาตนำออก</h3>
        <p className="wh-subtitle">
          หมดอายุ = วันที่ออก + 1 เดือน · ระบบจะผูกกับทุกเครื่องที่จัดสรรไว้สำหรับประเทศนี้
        </p>

        <div className="itc-form-grid">
          <label className="itc-field">
            <span>เลขใบอนุญาตนำออก</span>
            <input value={form.license_no} onChange={(e) => setForm({ ...form, license_no: e.target.value })} />
          </label>

          <label className="itc-field">
            <span>ประเทศปลายทาง</span>
            <SelectField
              value={form.country}
              onChange={(country) => setForm({ ...form, country })}
              placeholder="— เลือกประเทศ —"
              options={countries.map((c) => ({ value: c, label: c }))}
            />
          </label>

          <label className="itc-field">
            <span>ใบอนุญาตนำเข้าอ้างอิง</span>
            <SelectField
              value={form.import_license_no}
              onChange={(v) => setForm({ ...form, import_license_no: v })}
              placeholder="— อัตโนมัติจากเครื่อง —"
              options={[
                { value: '', label: '— อัตโนมัติจากเครื่อง —' },
                ...importLicenses.map((l) => ({ value: l.LicenseNo, label: l.LicenseNo })),
              ]}
            />
          </label>

          <label className="itc-field">
            <span>วันที่ออกใบอนุญาต</span>
            <input
              type="date"
              value={form.issue_date}
              onChange={(e) => setForm({ ...form, issue_date: e.target.value })}
            />
          </label>
        </div>

        <p className="wh-subtitle">จะผูกกับ {candidates.length} เครื่อง</p>

        <button className="wh-issue-btn" onClick={submit}>
          บันทึกใบอนุญาตนำออก
        </button>
      </section>

      <section className="itc-card">
        <h3 className="tsf-form-title">สแกนส่งออกจริง</h3>
        <p className="wh-subtitle">
          ระบบจะบล็อกทันทีถ้ายังไม่มีใบอนุญาต ใบหมดอายุ ประเทศไม่ตรง หรือเครื่องนี้ส่งออกไปแล้ว
        </p>
        <form className="itc-scan-row" onSubmit={handleExport}>
          <SelectField
            value={scanCountry}
            onChange={setScanCountry}
            placeholder="ไม่ระบุปลายทาง"
            options={[
              { value: '', label: 'ไม่ระบุปลายทาง' },
              ...countries.map((c) => ({ value: c, label: c })),
            ]}
          />
          <input
            className="wh-search"
            value={scan}
            onChange={(e) => setScan(e.target.value)}
            placeholder="ยิงบาร์โค้ดตอนของขึ้นรถ — ป้ายไหนก็ได้"
          />
          <button className="wh-issue-btn" type="submit">
            ยืนยันส่งออก
          </button>
        </form>
      </section>

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>ใบนำออก</th>
              <th>ประเทศ</th>
              <th>ใบนำเข้า</th>
              <th>จำนวน</th>
              <th>ส่งแล้ว</th>
              <th>หมดอายุ</th>
              <th>เหลือ (วัน)</th>
              <th>บัญชีแนบ</th>
            </tr>
          </thead>
          <tbody>
            {exportLicenses.map((l) => (
              <tr key={l.LicenseNo}>
                <td data-label="ใบนำออก" className="wh-cell-head">{l.LicenseNo}</td>
                <td data-label="ประเทศ">{l.Country}</td>
                <td data-label="ใบนำเข้า">{l.ImportLicenseNo}</td>
                <td data-label="จำนวน">{l.Qty}</td>
                <td data-label="ส่งแล้ว">{l.exported_count}</td>
                <td data-label="หมดอายุ">{fmtDate(l.ExpireDate)}</td>
                <td data-label="เหลือ (วัน)" className={l.days_left <= 3 ? 'itc-cell-bad' : ''}>{l.days_left}</td>
                <td data-label="บัญชีแนบ">
                  <button className="wh-issue-btn" onClick={() => downloadExportAttachment(l.LicenseNo)}>
                    ดาวน์โหลด
                  </button>
                </td>
              </tr>
            ))}
            {!exportLicenses.length && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
                  ยังไม่มีใบอนุญาตนำออก
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </>
  )
}

// ── Tab 5: Traceability ──────────────────────────────────────────────────────

function TraceTab({ onError }) {
  const [key, setKey] = useState('')
  const [data, setData] = useState(null)

  async function submit(e) {
    e.preventDefault()
    if (!key.trim()) return
    try {
      setData(await traceUnit(key.trim()))
    } catch (err) {
      setData(null)
      onError(err.message)
    }
  }

  const u = data?.unit

  return (
    <>
      <section className="itc-card">
        <h3 className="tsf-form-title">ค้นเลขเดียว เห็นทั้งเส้นทาง</h3>
        <form className="itc-scan-row" onSubmit={submit}>
          <input
            className="wh-search"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="IT Controller No. / IMEI / Serial No."
          />
          <button className="wh-issue-btn" type="submit">
            ค้นหา
          </button>
        </form>
      </section>

      {u && (
        <section className="itc-card">
          <div className="itc-trace-chain">
            {[
              ['Invoice', u.InvoiceNo],
              ['P.O.', u.PONo],
              ['ใบนำเข้า', u.ImportLicenseNo],
              ['ใบขนขาเข้า', u.DeclarationNo],
              ['ประเทศ', u.Country],
              ['ใบนำออก', u.ExportLicenseNo],
              ['เครื่องที่ประกอบ', data.machine_no],
            ].map(([label, value]) => (
              <div className="itc-trace-node" key={label}>
                <span className="itc-trace-label">{label}</span>
                <span className="itc-trace-value">{value || '—'}</span>
              </div>
            ))}
          </div>

          <dl className="itc-detail">
            <div>
              <dt>IT Controller No.</dt>
              <dd>{u.ITControllerNo}</dd>
            </div>
            <div>
              <dt>IMEI</dt>
              <dd>{u.IMEI}</dd>
            </div>
            <div>
              <dt>Serial No.</dt>
              <dd>{u.SerialNo}</dd>
            </div>
            <div>
              <dt>Part No.</dt>
              <dd>{u.PartNo}</dd>
            </div>
            <div>
              <dt>สถานะ</dt>
              <dd>{STATUS_LABEL[u.Status] || u.Status}</dd>
            </div>
            <div>
              <dt>ใบนำออกหมดอายุ</dt>
              <dd>{fmtDate(data.export_license?.ExpireDate)}</dd>
            </div>
          </dl>

          {!!data.documents?.length && (
            <div className="itc-doc-links">
              {data.documents.map((d) => (
                <a key={d.ID} href={d.FileURL} target="_blank" rel="noreferrer">
                  {d.DocType} {d.DocNo}
                </a>
              ))}
            </div>
          )}
        </section>
      )}
    </>
  )
}