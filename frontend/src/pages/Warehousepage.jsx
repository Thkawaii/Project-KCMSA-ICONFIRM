import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getWarehouseStock, issueWarehouseStock } from '../api/warehouse.js'
import { logout } from '../api/auth.js'

export default function WarehousePage() {
  const navigate = useNavigate()
  const [stock, setStock] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [search, setSearch] = useState('')
  const [issuingRow, setIssuingRow] = useState(null)
  const [issueQty, setIssueQty] = useState('')
  const [issueRemark, setIssueRemark] = useState('')
  const [issuing, setIssuing] = useState(false)
  const [issueError, setIssueError] = useState('')
  const [toast, setToast] = useState('')

  async function loadStock() {
    setLoading(true)
    setLoadError('')
    try {
      const data = await getWarehouseStock()
      setStock(data || [])
    } catch (err) {
      setLoadError(err.message || 'โหลดข้อมูลคลังไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadStock()
  }, [])

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    const rows = term
      ? stock.filter(
          (r) =>
            (r.OrderNo || '').toLowerCase().includes(term) ||
            (r.PartNo || '').toLowerCase().includes(term) ||
            (r.PartName || '').toLowerCase().includes(term)
        )
      : stock

    // เรียงตาม UploadDate (เก่าสุดก่อน) เพื่อบังคับใช้หลัก FIFO
    return [...rows].sort((a, b) => new Date(a.UploadDate) - new Date(b.UploadDate))
  }, [stock, search])

  function openIssueModal(row) {
    if (row.RemainQty <= 0) return
    setIssuingRow(row)
    setIssueQty('1')
    setIssueRemark('')
    setIssueError('')
  }

  function closeIssueModal() {
    setIssuingRow(null)
    setIssueQty('')
    setIssueRemark('')
    setIssueError('')
  }

  async function confirmIssue() {
    const qty = Number(issueQty)
    if (!qty || qty < 1 || qty > issuingRow.RemainQty) {
      setIssueError('จำนวนที่จ่ายไม่ถูกต้อง')
      return
    }

    setIssuing(true)
    setIssueError('')

    try {
      const result = await issueWarehouseStock(issuingRow.ID, qty, issueRemark)

      setToast(
        `จ่ายชิ้นส่วน ${issuingRow.PartName} (${issuingRow.PartNo}) จำนวน ${qty} ชิ้น อ้างอิง SO ${issuingRow.OrderNo} — เลขที่จ่ายของ ${result.stock_out_no}`
      )
      setTimeout(() => setToast(''), 5000)
      closeIssueModal()
      await loadStock()
    } catch (err) {
      setIssueError(err.message || 'จ่ายของไม่สำเร็จ')
    } finally {
      setIssuing(false)
    }
  }

  function handleLogout() {
    logout()
    navigate('/login')
  }

  return (
    <div className="wh-page">
      <header className="wh-topbar">
        <div className="brand-row">
          <span className="brand-badge">KOBELCO</span>
          <h1 className="brand-title-sm">I-CONFIRM</h1>
        </div>
        <button className="wh-logout-btn" onClick={handleLogout}>
          Logout
        </button>
      </header>

      <main className="wh-main">
        <div className="wh-heading-row">
          <div>
            <h2 className="wh-title">Warehouse — จ่ายชิ้นส่วน (Issue Parts)</h2>
            <p className="wh-subtitle">
              จ่ายชิ้นส่วนตามหลัก FIFO (First In, First Out) โดยอ้างอิง Sales Order ที่เกี่ยวข้อง
            </p>
          </div>
          <input
            className="wh-search"
            type="text"
            placeholder="ค้นหาด้วย SO / Part No / Part Name"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        {toast && <div className="wh-toast">{toast}</div>}
        {loadError && (
          <p className="form-error" role="alert">
            {loadError}
          </p>
        )}

        <div className="wh-table-card">
          <table className="wh-table">
            <thead>
              <tr>
                <th>FIFO #</th>
                <th>Sales Order</th>
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
                  <td colSpan={10} className="wh-empty-cell">
                    กำลังโหลดข้อมูล...
                  </td>
                </tr>
              )}
              {!loading &&
                filtered.map((row, idx) => (
                  <tr key={row.ID} className={row.RemainQty <= 0 ? 'wh-row-empty' : ''}>
                    <td>#{idx + 1}</td>
                    <td>{row.OrderNo}</td>
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
                  <td colSpan={10} className="wh-empty-cell">
                    ไม่พบข้อมูล
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </main>

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
                {issuing ? 'กำลังบันทึก...' : 'ยืนยันจ่ายของ'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}