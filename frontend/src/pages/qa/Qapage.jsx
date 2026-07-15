import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'

// TODO: mock ข้อมูล master + คิวรอตรวจของ QA ไว้ก่อน — ของจริงต้องดึงจาก backend
// (ตาราง QA / QA_CONFIRM ใน ERD) โดยคิวนี้ควรมาจากสิ่งที่ TSF Operator ส่งเข้ามา
const masterData = [
  { part_no: 'CV-1002', spec_code: 'SPEC-CV-A1' },
  { part_no: 'SM-3301', spec_code: 'SPEC-SM-B2' },
  { part_no: 'MP-4410', spec_code: 'SPEC-MP-C3' },
  { part_no: 'PA-5501', spec_code: 'SPEC-PA-D4' },
]

const initialQueue = [
  {
    id: 1,
    serial_number: 'SN-88231',
    expected_part_no: 'CV-1002',
    actual_part_no: 'CV-1002',
    expected_spec: 'SPEC-CV-A1',
    actual_spec: 'SPEC-CV-A1',
    result: null,
    remark: '',
  },
  {
    id: 2,
    serial_number: 'SN-88240',
    expected_part_no: 'SM-3301',
    actual_part_no: 'SM-3301',
    expected_spec: 'SPEC-SM-B2',
    actual_spec: 'SPEC-SM-B9', // ผิด spec โดยตั้งใจ เพื่อโชว์เคส FAIL
    result: null,
    remark: '',
  },
  {
    id: 3,
    serial_number: 'SN-88255',
    expected_part_no: 'MP-4410',
    actual_part_no: 'MP-4410',
    expected_spec: 'SPEC-MP-C3',
    actual_spec: 'SPEC-MP-C3',
    result: null,
    remark: '',
  },
]

export default function QAPage() {
  const navigate = useNavigate()
  const [queue, setQueue] = useState(initialQueue)
  const [auditLog, setAuditLog] = useState([])
  const [remarkDraft, setRemarkDraft] = useState({})

  const pendingCount = useMemo(() => queue.filter((q) => q.result === null).length, [queue])

  function isMismatch(row) {
    return row.expected_part_no !== row.actual_part_no || row.expected_spec !== row.actual_spec
  }

  function confirmResult(row, result) {
    const remark = remarkDraft[row.id] ?? ''

    setQueue((prev) =>
      prev.map((q) => (q.id === row.id ? { ...q, result, remark } : q))
    )

    setAuditLog((prev) => [
      {
        id: `${row.id}-${Date.now()}`,
        source_table: 'QA_CONFIRM',
        action: result === 'PASS' ? 'Confirm PASS' : 'Confirm FAIL',
        serial_number: row.serial_number,
        remark,
        action_datetime: new Date().toLocaleString('th-TH'),
      },
      ...prev,
    ])
  }

  function handleLogout() {
    localStorage.removeItem('iconfirm_token')
    localStorage.removeItem('iconfirm_role')
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
            <h2 className="wh-title">QA — ตรวจสอบและยืนยันข้อมูล</h2>
            <p className="wh-subtitle">
              เทียบข้อมูลจริง (Actual) กับ Master Data — ตรวจ P/N, S/N และ Spec Code ก่อนยืนยันผล
              {pendingCount > 0 && <> · รอตรวจ {pendingCount} รายการ</>}
            </p>
          </div>
        </div>

        <div className="wh-table-card">
          <table className="wh-table">
            <thead>
              <tr>
                <th>S/N</th>
                <th>Expected P/N</th>
                <th>Actual P/N</th>
                <th>Expected Spec</th>
                <th>Actual Spec</th>
                <th>ผลเทียบ</th>
                <th>Remark</th>
                <th>ผลยืนยัน</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {queue.map((row) => {
                const mismatch = isMismatch(row)
                return (
                  <tr key={row.id}>
                    <td>{row.serial_number}</td>
                    <td>{row.expected_part_no}</td>
                    <td className={mismatch && row.expected_part_no !== row.actual_part_no ? 'qa-cell-fail' : ''}>
                      {row.actual_part_no}
                    </td>
                    <td>{row.expected_spec}</td>
                    <td className={mismatch && row.expected_spec !== row.actual_spec ? 'qa-cell-fail' : ''}>
                      {row.actual_spec}
                    </td>
                    <td>
                      <span className={mismatch ? 'wh-qty-badge wh-qty-zero' : 'wh-qty-badge'}>
                        {mismatch ? 'ไม่ตรง' : 'ตรงกัน'}
                      </span>
                    </td>
                    <td>
                      <input
                        className="qa-remark-input"
                        type="text"
                        placeholder="หมายเหตุ (ถ้ามี)"
                        disabled={row.result !== null}
                        value={remarkDraft[row.id] ?? row.remark}
                        onChange={(e) =>
                          setRemarkDraft((prev) => ({ ...prev, [row.id]: e.target.value }))
                        }
                      />
                    </td>
                    <td>
                      {row.result ? (
                        <span
                          className={
                            row.result === 'PASS' ? 'qa-result-badge qa-pass' : 'qa-result-badge qa-fail'
                          }
                        >
                          {row.result}
                        </span>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="qa-action-cell">
                      <button
                        className="qa-pass-btn"
                        disabled={row.result !== null}
                        onClick={() => confirmResult(row, 'PASS')}
                      >
                        PASS
                      </button>
                      <button
                        className="qa-fail-btn"
                        disabled={row.result !== null}
                        onClick={() => confirmResult(row, 'FAIL')}
                      >
                        FAIL
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>

        <h3 className="tsf-form-title qa-audit-heading">Audit Log</h3>
        <div className="wh-table-card">
          <table className="wh-table">
            <thead>
              <tr>
                <th>S/N</th>
                <th>Action</th>
                <th>Remark</th>
                <th>เวลา</th>
              </tr>
            </thead>
            <tbody>
              {auditLog.map((log) => (
                <tr key={log.id}>
                  <td>{log.serial_number}</td>
                  <td>{log.action}</td>
                  <td>{log.remark || '—'}</td>
                  <td>{log.action_datetime}</td>
                </tr>
              ))}
              {auditLog.length === 0 && (
                <tr>
                  <td colSpan={4} className="wh-empty-cell">
                    ยังไม่มีการยืนยันผล
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </main>
    </div>
  )
}