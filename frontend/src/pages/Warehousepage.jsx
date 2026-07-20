import { useEffect, useMemo, useState } from 'react'
import { getWarehouseStock, issueWarehouseStock, getWhConfirms, uploadWarehouseStock } from '../api/warehouse.js'
import AppShell from '../components/AppShell.jsx'
import BarcodeScannerModal from '../components/BarcodeScannerModal.jsx'

const navItems = [{ to: '/warehouse', label: 'Issue Parts', icon: '📦' }]

const STOCK_TABS = [
  { key: 'all', label: 'ทั้งหมด' },
  { key: 'not_issued', label: 'ยังไม่เคยจ่าย' },
  { key: 'issued', label: 'จ่ายแล้ว' },
]

export default function WarehousePage() {
  const [pageTab, setPageTab] = useState('issue') // 'issue' | 'confirm'
  const [stock, setStock] = useState([])
  const [transfers, setTransfers] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [search, setSearch] = useState('')
  const [selectedSO, setSelectedSO] = useState('')
  const [stockTab, setStockTab] = useState('all')

  const [issuingRow, setIssuingRow] = useState(null)
  const [issueQty, setIssueQty] = useState('')
  const [scanPartNo, setScanPartNo] = useState('')
  const [scanSerial, setScanSerial] = useState('')
  const [issueRemark, setIssueRemark] = useState('')
  const [issuing, setIssuing] = useState(false)
  const [issueError, setIssueError] = useState('')
  const [toast, setToast] = useState('')
  const [scanningField, setScanningField] = useState(null) // 'part' | 'serial' | null

  // ===== อัปโหลด SO / สต็อกเข้าคลัง =====
  const [soFile, setSoFile] = useState(null)
  const [soUploading, setSoUploading] = useState(false)
  const [soMsg, setSoMsg] = useState(null)

  // ===== Part Confirmation dashboard (ตาราง WHConfirm) =====
  const [partFilter, setPartFilter] = useState('')
  const [machineFilter, setMachineFilter] = useState('')
  const [dateTab, setDateTab] = useState('all') // all | day | week | month
  const [pcSearch, setPcSearch] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [page, setPage] = useState(1)

  async function loadAll() {
    setLoading(true)
    setLoadError('')
    try {
      const [stockData, transferData] = await Promise.all([getWarehouseStock(), getWhConfirms()])
      setStock(stockData || [])
      setTransfers(transferData || [])
    } catch (err) {
      setLoadError(err.message || 'โหลดข้อมูลคลังไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadAll()
  }, [])

  useEffect(() => {
    setPage(1)
  }, [partFilter, machineFilter, dateTab, pcSearch, pageSize])

  async function handleSoFile(e) {
    const file = e.target.files?.[0]
    if (!file) return
    setSoFile(file)
  }

  async function handleSoUpload() {
    if (!soFile) {
      setSoMsg({ error: 'กรุณาเลือกไฟล์ Excel ก่อน' })
      return
    }
    setSoUploading(true)
    setSoMsg(null)
    try {
      const result = await uploadWarehouseStock(soFile)
      setSoMsg({ success: `นำเข้าสำเร็จ ${result.imported} รายการ` })
      setSoFile(null)
      await loadAll()
    } catch (err) {
      setSoMsg({ error: err.message || 'อัปโหลดไม่สำเร็จ' })
    } finally {
      setSoUploading(false)
    }
  }

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    let rows = term
      ? stock.filter(
          (r) =>
            (r.OrderNo || '').toLowerCase().includes(term) ||
            (r.PartNo || '').toLowerCase().includes(term) ||
            (r.PartName || '').toLowerCase().includes(term)
        )
      : stock

    if (selectedSO) rows = rows.filter((r) => r.OrderNo === selectedSO)

    if (stockTab === 'issued') rows = rows.filter((r) => Boolean(r.StockOutNo))
    if (stockTab === 'not_issued') rows = rows.filter((r) => !r.StockOutNo)

    // เรียงตาม UploadDate (เก่าสุดก่อน) เพื่อบังคับใช้หลัก FIFO
    return [...rows].sort((a, b) => new Date(a.UploadDate) - new Date(b.UploadDate))
  }, [stock, search, selectedSO, stockTab])

  // รายชื่อ S/O ทั้งหมด พร้อมสรุปว่าจ่ายไปแล้วกี่รายการ/ทั้งหมดกี่รายการ —
  // ใช้ตอบโจทย์ "WH: supply following FIFO & S/O" ให้เลือก S/O ก่อนแล้วดู
  // เฉพาะของที่ต้องจ่ายให้ S/O นั้น เรียง FIFO ในตัว
  const salesOrders = useMemo(() => {
    const map = new Map()
    for (const r of stock) {
      if (!r.OrderNo) continue
      if (!map.has(r.OrderNo)) map.set(r.OrderNo, { total: 0, issued: 0 })
      const entry = map.get(r.OrderNo)
      entry.total += 1
      if (r.StockOutNo) entry.issued += 1
    }
    return Array.from(map.entries()).map(([orderNo, v]) => ({ orderNo, ...v }))
  }, [stock])

  const filteredTransfers = useMemo(() => {
    let rows = transfers

    if (partFilter) rows = rows.filter((t) => t.PartName === partFilter)
    if (machineFilter) rows = rows.filter((t) => t.MachineModel === machineFilter)

    if (dateTab !== 'all') {
      const now = new Date()
      rows = rows.filter((t) => {
        const diffDays = (now - new Date(t.ConfirmDatetime)) / (1000 * 60 * 60 * 24)
        if (dateTab === 'day') return diffDays <= 1
        if (dateTab === 'week') return diffDays <= 7
        if (dateTab === 'month') return diffDays <= 31
        return true
      })
    }

    const term = pcSearch.trim().toLowerCase()
    if (term) {
      rows = rows.filter(
        (t) =>
          (t.PartNo || '').toLowerCase().includes(term) ||
          (t.PartName || '').toLowerCase().includes(term) ||
          (t.MachineModel || '').toLowerCase().includes(term) ||
          (t.OrderNo || '').toLowerCase().includes(term) ||
          (t.Name || '').toLowerCase().includes(term)
      )
    }

    return [...rows].sort((a, b) => new Date(b.ConfirmDatetime) - new Date(a.ConfirmDatetime))
  }, [transfers, partFilter, machineFilter, dateTab, pcSearch])

  const partOptions = useMemo(
    () => Array.from(new Set(transfers.map((t) => t.PartName).filter(Boolean))),
    [transfers]
  )
  const machineOptions = useMemo(
    () => Array.from(new Set(transfers.map((t) => t.MachineModel).filter(Boolean))),
    [transfers]
  )

  const pcTotalPages = Math.max(1, Math.ceil(filteredTransfers.length / pageSize))
  const pagedTransfers = filteredTransfers.slice((page - 1) * pageSize, page * pageSize)
  function goToPage(p) {
    setPage(Math.min(Math.max(1, p), pcTotalPages))
  }

  const stockCounts = useMemo(
    () => ({
      all: stock.length,
      issued: stock.filter((r) => Boolean(r.StockOutNo)).length,
      not_issued: stock.filter((r) => !r.StockOutNo).length,
    }),
    [stock]
  )

  const transferCounts = useMemo(
    () => ({
      all: transfers.length,
      SENT: transfers.filter((t) => t.TransferStatus === 'SENT').length,
      RECEIVED: transfers.filter((t) => t.TransferStatus === 'RECEIVED').length,
    }),
    [transfers]
  )

  function openIssueModal(row) {
    if (row.RemainQty <= 0) return
    setIssuingRow(row)
    setIssueQty('1')
    setScanPartNo('')
    setScanSerial('')
    setIssueRemark('')
    setIssueError('')
  }

  function handleScanned(value) {
    if (scanningField === 'part') setScanPartNo(value)
    if (scanningField === 'serial') setScanSerial(value)
    setScanningField(null)
  }

  function closeIssueModal() {
    setIssuingRow(null)
    setIssueQty('')
    setScanPartNo('')
    setScanSerial('')
    setIssueRemark('')
    setIssueError('')
  }

  async function confirmIssue() {
    const qty = Number(issueQty)
    if (!qty || qty < 1 || qty > issuingRow.RemainQty) {
      setIssueError('จำนวนที่จ่ายไม่ถูกต้อง')
      return
    }

    if (!scanPartNo.trim()) {
      setIssueError('กรุณา Scan P/N ก่อนยืนยัน')
      return
    }

    if (scanPartNo.trim() !== issuingRow.PartNo) {
      setIssueError(`Scan P/N ไม่ตรงกับรายการนี้ (สแกนได้ "${scanPartNo.trim()}" แต่รายการคือ "${issuingRow.PartNo}")`)
      return
    }

    if (!scanSerial.trim()) {
      setIssueError('กรุณา Scan S/N ก่อนยืนยัน')
      return
    }

    setIssuing(true)
    setIssueError('')

    try {
      const result = await issueWarehouseStock(issuingRow.ID, {
        qty,
        scannedPartNo: scanPartNo.trim(),
        scannedSerial: scanSerial.trim(),
        remark: issueRemark,
      })

      setToast(
        `จ่ายชิ้นส่วน ${issuingRow.PartName} (${issuingRow.PartNo}) จำนวน ${qty} ชิ้น อ้างอิง SO ${issuingRow.OrderNo} — เลขที่จ่ายของ ${result.stock_out_no} — ส่งให้ TSF แล้ว รอรับของ`
      )
      setTimeout(() => setToast(''), 6000)
      closeIssueModal()
      await loadAll()
    } catch (err) {
      setIssueError(err.message || 'จ่ายของไม่สำเร็จ')
    } finally {
      setIssuing(false)
    }
  }

  return (
    <>
    <AppShell navItems={navItems} roleLabel="Warehouse">
        <div className="wh-heading-row">
          <div>
            <h2 className="wh-title">Warehouse</h2>
            <p className="wh-subtitle">เลือกโหมดการทำงาน</p>
          </div>
        </div>

        <div className="vr-tabs" style={{ marginBottom: 24 }}>
          <button
            className={'vr-tab' + (pageTab === 'issue' ? ' vr-tab-active' : '')}
            onClick={() => setPageTab('issue')}
          >
            📦 จ่ายของ (FIFO &amp; S/O)
          </button>
          <button
            className={'vr-tab' + (pageTab === 'confirm' ? ' vr-tab-active' : '')}
            onClick={() => setPageTab('confirm')}
          >
            ✅ Part Confirmation
          </button>
        </div>

        {toast && <div className="wh-toast">{toast}</div>}
        {loadError && (
          <p className="form-error" role="alert">
            {loadError}
          </p>
        )}

    {pageTab === 'issue' && (
      <>
        <div className="upload-card" style={{ maxWidth: 320, marginBottom: 24, textAlign: 'left', padding: 14 }}>
          <p className="wh-subtitle" style={{ margin: '0 0 8px', fontSize: 12 }}>
            อัปโหลด SO / สต็อกเข้าคลัง (Excel)
          </p>
          <label className={'upload-dropzone' + (soFile ? ' upload-dropzone-filled' : '')} htmlFor="soFile" style={{ padding: 12 }}>
            <input id="soFile" type="file" accept=".xlsx,.xls" onChange={handleSoFile} className="upload-card-input-hidden" />
            <UploadCloudIcon />
            <span className="upload-dropzone-text">{soFile ? soFile.name : 'คลิกเพื่อเลือกไฟล์ Excel'}</span>
          </label>
          <button className="wh-issue-btn upload-card-btn" onClick={handleSoUpload} disabled={soUploading}>
            {soUploading ? 'กำลังอัปโหลด...' : 'อัปโหลด'}
          </button>
          {soMsg?.success && <p className="upload-card-msg upload-card-msg-ok">{soMsg.success}</p>}
          {soMsg?.error && <p className="upload-card-msg upload-card-msg-err">{soMsg.error}</p>}
        </div>

        {!selectedSO ? (
          <div className="tsf-form-card" style={{ maxWidth: 420 }}>
            <h3 className="tsf-form-title">1. เลือก Sales Order ก่อนเริ่มจ่ายของ</h3>
            <p className="wh-subtitle" style={{ marginBottom: 14 }}>
              ระบบจะโชว์เฉพาะของที่ต้องจ่ายให้ S/O นี้ เรียง FIFO (เก่าสุดก่อน) ให้อัตโนมัติ
            </p>
            <select
              className="wh-modal-input"
              value={selectedSO}
              onChange={(e) => setSelectedSO(e.target.value)}
            >
              <option value="">-- เลือก Sales Order --</option>
              {salesOrders.map((so) => (
                <option key={so.orderNo} value={so.orderNo}>
                  {so.orderNo} ({so.issued}/{so.total} จ่ายแล้ว)
                </option>
              ))}
            </select>
            {salesOrders.length === 0 && (
              <p className="wh-subtitle" style={{ marginTop: 10 }}>
                ยังไม่มี Sales Order ในระบบ — อัปโหลดไฟล์ Excel ด้านบนก่อน
              </p>
            )}
          </div>
        ) : (
          <>
            <div className="wh-heading-row">
              <div>
                <h3 className="tsf-form-title" style={{ marginBottom: 2 }}>
                  2. จ่ายของสำหรับ S/O: {selectedSO}
                </h3>
                <p className="wh-subtitle">
                  เรียง FIFO (เก่าสุดก่อน) ·{' '}
                  {salesOrders.find((s) => s.orderNo === selectedSO)?.issued || 0}/
                  {salesOrders.find((s) => s.orderNo === selectedSO)?.total || 0} รายการจ่ายแล้ว
                </p>
              </div>
              <div style={{ display: 'flex', gap: 10 }}>
                <input
                  className="wh-search"
                  type="text"
                  placeholder="ค้นหาด้วย Part No / Part Name"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
                <button className="wh-modal-cancel" onClick={() => setSelectedSO('')}>
                  เปลี่ยน S/O
                </button>
              </div>
            </div>

            <div className="vr-tabs">
              {STOCK_TABS.map((tab) => (
                <button
                  key={tab.key}
                  className={'vr-tab' + (stockTab === tab.key ? ' vr-tab-active' : '')}
                  onClick={() => setStockTab(tab.key)}
                >
                  {tab.label} ({stockCounts[tab.key]})
                </button>
              ))}
            </div>

            <div className="wh-table-card">
              <table className="wh-table">
                <thead>
                  <tr>
                    <th>FIFO #</th>
                    <th>Part No.</th>
                    <th>Part Name</th>
                    <th>Assembly For</th>
                    <th>Shelf</th>
                    <th>Remain Qty</th>
                    <th>Upload Date</th>
                    <th>Stock Out No.</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {loading && (
                    <tr>
                      <td colSpan={9} className="wh-empty-cell">
                        กำลังโหลดข้อมูล...
                      </td>
                    </tr>
                  )}
                  {!loading &&
                    filtered.map((row, idx) => (
                      <tr key={row.ID} className={row.RemainQty <= 0 ? 'wh-row-empty' : ''}>
                        <td>#{idx + 1}</td>
                        <td>{row.PartNo}</td>
                        <td>{row.PartName}</td>
                        <td>{row.AssemblyPartName}</td>
                        <td>
                          {row.Shelf1}-{row.Shelf2}
                        </td>
                        <td>
                          <span className={row.RemainQty <= 0 ? 'wh-qty-badge wh-qty-zero' : 'wh-qty-badge'}>
                            {row.RemainQty}
                          </span>
                        </td>
                        <td>{row.UploadDate ? new Date(row.UploadDate).toLocaleDateString('th-TH') : '—'}</td>
                        <td>{row.StockOutNo || '—'}</td>
                        <td>
                          <button
                            className="wh-issue-btn"
                            disabled={row.RemainQty <= 0}
                            onClick={() => openIssueModal(row)}
                          >
                            จ่ายของ
                          </button>
                        </td>
                      </tr>
                    ))}
                  {!loading && filtered.length === 0 && (
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
        )}
      </>
    )}

    {pageTab === 'confirm' && (
      <>
        <div className="wh-heading-row">
          <div>
            <h2 className="wh-title" style={{ fontSize: 19 }}>
              Part Confirmation
            </h2>
            <p className="wh-subtitle">รายการที่จ่ายแล้ว — รอ TSF กด "รับอะไหล่" ที่หน้า TSF Operator</p>
          </div>
        </div>

        <div className="tsf-form-card" style={{ marginBottom: 20 }}>
          <label className="wh-modal-label">เลือกพาร์ท</label>
          <select className="wh-modal-input" value={partFilter} onChange={(e) => setPartFilter(e.target.value)}>
            <option value="">-- ทั้งหมด --</option>
            {partOptions.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>

          <label className="wh-modal-label">เลือก Machine No.</label>
          <select className="wh-modal-input" value={machineFilter} onChange={(e) => setMachineFilter(e.target.value)}>
            <option value="">-- ทั้งหมด --</option>
            {machineOptions.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        </div>

        <div className="tsf-stats-row">
          <div className="tsf-stat-card">
            <span className="tsf-stat-icon">📦</span>
            <div>
              <div className="tsf-stat-value">{transferCounts.all}</div>
              <div className="tsf-stat-label">ทั้งหมด</div>
            </div>
          </div>
          <div className="tsf-stat-card">
            <span className="tsf-stat-icon">🚚</span>
            <div>
              <div className="tsf-stat-value">{transferCounts.SENT}</div>
              <div className="tsf-stat-label">รอ TSF รับ</div>
            </div>
          </div>
          <div className="tsf-stat-card">
            <span className="tsf-stat-icon">✅</span>
            <div>
              <div className="tsf-stat-value">{transferCounts.RECEIVED}</div>
              <div className="tsf-stat-label">TSF รับแล้ว</div>
            </div>
          </div>
          <div className="tsf-stat-card">
            <span className="tsf-stat-icon">🧾</span>
            <div>
              <div className="tsf-stat-value">{salesOrders.length}</div>
              <div className="tsf-stat-label">Sales Order ที่มีการจ่าย</div>
            </div>
          </div>
        </div>

        <div className="wh-heading-row">
          <div />
          <div className="vr-tabs">
            {[
              { key: 'all', label: 'ทั้งหมด' },
              { key: 'day', label: 'รายวัน' },
              { key: 'week', label: 'รายสัปดาห์' },
              { key: 'month', label: 'รายเดือน' },
            ].map((tab) => (
              <button
                key={tab.key}
                className={'vr-tab' + (dateTab === tab.key ? ' vr-tab-active' : '')}
                onClick={() => setDateTab(tab.key)}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        <div className="tsf-history-toolbar">
          <label className="tsf-history-pagesize">
            <select value={pageSize} onChange={(e) => setPageSize(Number(e.target.value))}>
              <option value={10}>10</option>
              <option value={25}>25</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
            </select>
            entries per page
          </label>
          <input
            className="wh-search"
            type="text"
            placeholder="ค้นหา Part No / Machine No / SO / ผู้จ่าย"
            value={pcSearch}
            onChange={(e) => setPcSearch(e.target.value)}
          />
        </div>

        <div className="wh-table-card">
          <table className="wh-table">
            <thead>
              <tr>
                <th>Sales Order</th>
                <th>Part No.</th>
                <th>Part Name</th>
                <th>Machine No.</th>
                <th>จ่ายโดย</th>
                <th>เวลาจ่าย</th>
                <th>สถานะ</th>
              </tr>
            </thead>
            <tbody>
              {!loading &&
                pagedTransfers.map((t) => (
                  <tr key={t.ID}>
                    <td>{t.OrderNo || '—'}</td>
                    <td>{t.PartNo}</td>
                    <td>{t.PartName}</td>
                    <td>{t.MachineModel || '—'}</td>
                    <td>{t.Name}</td>
                    <td>{new Date(t.ConfirmDatetime).toLocaleString('th-TH')}</td>
                    <td>
                      <span
                        className={
                          t.TransferStatus === 'RECEIVED' ? 'wh-qty-badge' : 'wh-qty-badge wh-qty-zero'
                        }
                      >
                        {t.TransferStatus === 'RECEIVED' ? 'TSF รับแล้ว' : 'รอ TSF รับ'}
                      </span>
                    </td>
                  </tr>
                ))}
              {!loading && pagedTransfers.length === 0 && (
                <tr>
                  <td colSpan={7} className="wh-empty-cell">
                    ไม่พบรายการ
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {!loading && filteredTransfers.length > 0 && (
          <div className="tsf-pagination">
            <span className="wh-subtitle" style={{ fontSize: 13 }}>
              Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, filteredTransfers.length)} of{' '}
              {filteredTransfers.length} entries
            </span>
            <div className="tsf-pagination-buttons">
              <button className="wh-modal-cancel" onClick={() => goToPage(1)} disabled={page === 1}>
                «
              </button>
              <button className="wh-modal-cancel" onClick={() => goToPage(page - 1)} disabled={page === 1}>
                ‹
              </button>
              <span className="tsf-pagination-current">
                {page} / {pcTotalPages}
              </span>
              <button className="wh-modal-cancel" onClick={() => goToPage(page + 1)} disabled={page === pcTotalPages}>
                ›
              </button>
              <button className="wh-modal-cancel" onClick={() => goToPage(pcTotalPages)} disabled={page === pcTotalPages}>
                »
              </button>
            </div>
          </div>
        )}
      </>
    )}
    </AppShell>

      {issuingRow && (
        <div className="wh-modal-overlay" onClick={closeIssueModal}>
          <div className="wh-modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="wh-modal-title">ยืนยันการจ่ายชิ้นส่วน</h3>
            <p className="wh-modal-line">
              <strong>{issuingRow.PartName}</strong> ({issuingRow.PartNo})
            </p>
            <p className="wh-modal-line">Sales Order: {issuingRow.OrderNo}</p>
            <p className="wh-modal-line">คงเหลือ: {issuingRow.RemainQty} ชิ้น</p>

            <label className="wh-modal-label" htmlFor="issueQty">
              จำนวนที่จะจ่าย
            </label>
            <input
              id="issueQty"
              type="number"
              min="1"
              max={issuingRow.RemainQty}
              className="wh-modal-input"
              value={issueQty}
              onChange={(e) => setIssueQty(e.target.value)}
              autoFocus
            />

            <label className="wh-modal-label" htmlFor="scanPartNo">
              Scan P/N (ต้องตรงกับ {issuingRow.PartNo})
            </label>
            <div className="scan-input-row">
              <input
                id="scanPartNo"
                type="text"
                className="wh-modal-input"
                placeholder="สแกนหรือกรอก P/N"
                value={scanPartNo}
                onChange={(e) => setScanPartNo(e.target.value)}
              />
              <button
                type="button"
                className="scan-icon-btn"
                aria-label="เปิดกล้องสแกน P/N"
                onClick={() => setScanningField('part')}
              >
                <CameraIcon />
              </button>
            </div>

            <label className="wh-modal-label" htmlFor="scanSerial">
              Scan S/N
            </label>
            <div className="scan-input-row">
              <input
                id="scanSerial"
                type="text"
                className="wh-modal-input"
                placeholder="สแกนหรือกรอก S/N"
                value={scanSerial}
                onChange={(e) => setScanSerial(e.target.value)}
              />
              <button
                type="button"
                className="scan-icon-btn"
                aria-label="เปิดกล้องสแกน S/N"
                onClick={() => setScanningField('serial')}
              >
                <CameraIcon />
              </button>
            </div>

            <label className="wh-modal-label" htmlFor="issueRemark">
              หมายเหตุ (ถ้ามี)
            </label>
            <input
              id="issueRemark"
              type="text"
              className="wh-modal-input"
              value={issueRemark}
              onChange={(e) => setIssueRemark(e.target.value)}
            />

            {issueError && (
              <p className="form-error" role="alert">
                {issueError}
              </p>
            )}

            <div className="wh-modal-actions">
              <button className="wh-modal-cancel" onClick={closeIssueModal} disabled={issuing}>
                ยกเลิก
              </button>
              <button className="wh-modal-confirm" onClick={confirmIssue} disabled={issuing}>
                {issuing ? 'กำลังบันทึก...' : 'ยืนยันจ่ายของ + ส่งให้ TSF'}
              </button>
            </div>
          </div>
        </div>
      )}

      {scanningField && (
        <BarcodeScannerModal
          title={scanningField === 'part' ? 'สแกน Part No.' : 'สแกน Serial No.'}
          onDetected={handleScanned}
          onClose={() => setScanningField(null)}
        />
      )}
    </>
  )
}

function UploadCloudIcon() {
  return (
    <svg width="26" height="26" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M7 18a4.5 4.5 0 01-.4-8.98A5.5 5.5 0 0117 8.5a4 4 0 01-.5 7.98"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M12 12v7m0-7l-2.5 2.5M12 12l2.5 2.5"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function CameraIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M4 8a2 2 0 012-2h1.2l.8-1.4A1 1 0 018.86 4h6.28a1 1 0 01.86.6L16.8 6H18a2 2 0 012 2v9a2 2 0 01-2 2H6a2 2 0 01-2-2V8z"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="13" r="3.2" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  )
}