import { useEffect, useState } from 'react'
import { getWhConfirms, receiveWhTransfer } from '../api/warehouse.js'
import AppShell from '../components/AppShell.jsx'

const navItems = [
  { to: '/tsf', label: 'Scan & Validate', icon: '⇄' },
  { to: '/tsf/receive', label: 'Receive Material', icon: '📥' },
]

export default function TSFReceivePage() {
  const [pendingTransfers, setPendingTransfers] = useState([])
  const [receivingId, setReceivingId] = useState(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')

  async function loadPending() {
    setLoading(true)
    setLoadError('')
    try {
      const pending = await getWhConfirms('SENT')
      setPendingTransfers(pending || [])
    } catch (err) {
      setLoadError(err.message || 'โหลดข้อมูลไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadPending()
  }, [])

  async function handleReceive(id) {
    setReceivingId(id)
    try {
      await receiveWhTransfer(id)
      await loadPending()
    } catch (err) {
      setLoadError(err.message || 'รับอะไหล่ไม่สำเร็จ')
    } finally {
      setReceivingId(null)
    }
  }

  return (
    <AppShell navItems={navItems} roleLabel="MFG">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">รับอะไหล่จาก Warehouse</h2>
          <p className="wh-subtitle">
            กดรับอะไหล่ที่ Warehouse จ่ายมาก่อนเริ่มสแกนติดตั้ง
            {pendingTransfers.length > 0 && <> · รอรับ {pendingTransfers.length} รายการ</>}
          </p>
        </div>
      </div>

      {loadError && (
        <p className="form-error" role="alert">
          {loadError}
        </p>
      )}

      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Sales Order</th>
              <th>Part No.</th>
              <th>S/N</th>
              <th>Part Name</th>
              <th>จ่ายโดย</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={6} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              pendingTransfers.map((t) => (
                <tr key={t.ID}>
                  <td className="wh-cell-head" data-label="Sales Order">{t.OrderNo || '—'}</td>
                  <td data-label="Part No.">{t.PartNo}</td>
                  <td data-label="S/N">{t.SerialNo || '—'}</td>
                  <td data-label="Part Name">{t.PartName}</td>
                  <td data-label="จ่ายโดย">{t.Name}</td>
                  <td className="wh-cell-action">
                    <button
                      className="wh-issue-btn"
                      disabled={receivingId === t.ID}
                      onClick={() => handleReceive(t.ID)}
                    >
                      {receivingId === t.ID ? 'กำลังรับ...' : 'รับอะไหล่'}
                    </button>
                  </td>
                </tr>
              ))}
            {!loading && pendingTransfers.length === 0 && (
              <tr>
                <td colSpan={6} className="wh-empty-cell">
                  ไม่มีอะไหล่รอรับ
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </AppShell>
  )
}