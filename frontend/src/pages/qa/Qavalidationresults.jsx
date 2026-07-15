import { useOutletContext } from 'react-router-dom'

export default function QAValidationResults() {
  const { queue, auditLog, remarkDraft, setRemarkDraft, isMismatch, confirmResult, loading, loadError } =
    useOutletContext()

  const pendingCount = queue.filter((q) => !q.Result).length

  return (
    <>
      <div className="qa-page-heading">
        <h2 className="wh-title">Validation Results</h2>
        <p className="wh-subtitle">
          เทียบข้อมูลจริง (Actual) กับ Master Data — ตรวจ P/N, S/N และ Spec Code ก่อนยืนยันผล
          {pendingCount > 0 && <> · รอตรวจ {pendingCount} รายการ</>}
        </p>
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
            {loading && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
                  กำลังโหลดข้อมูล...
                </td>
              </tr>
            )}
            {!loading &&
              queue.map((row) => {
                const mismatch = isMismatch(row)
                const confirmed = Boolean(row.Result)
                return (
                  <tr key={row.ID}>
                    <td>{row.ExpectedPartNo}</td>
                    <td className={mismatch && row.ExpectedPartNo !== row.ActualPartNo ? 'qa-cell-fail' : ''}>
                      {row.ActualPartNo}
                    </td>
                    <td>{row.ExpectedSpec}</td>
                    <td className={mismatch && row.ExpectedSpec !== row.ActualSpec ? 'qa-cell-fail' : ''}>
                      {row.ActualSpec}
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
                        disabled={confirmed}
                        value={remarkDraft[row.ID] ?? ''}
                        onChange={(e) =>
                          setRemarkDraft((prev) => ({ ...prev, [row.ID]: e.target.value }))
                        }
                      />
                    </td>
                    <td>
                      {row.Result ? (
                        <span
                          className={
                            row.Result === 'PASS' ? 'qa-result-badge qa-pass' : 'qa-result-badge qa-fail'
                          }
                        >
                          {row.Result}
                        </span>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="qa-action-cell">
                      <button
                        className="qa-pass-btn"
                        disabled={confirmed}
                        onClick={() => confirmResult(row, 'PASS')}
                      >
                        PASS
                      </button>
                      <button
                        className="qa-fail-btn"
                        disabled={confirmed}
                        onClick={() => confirmResult(row, 'FAIL')}
                      >
                        FAIL
                      </button>
                    </td>
                  </tr>
                )
              })}
            {!loading && queue.length === 0 && (
              <tr>
                <td colSpan={8} className="wh-empty-cell">
                  ไม่มีรายการรอตรวจ
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <h3 className="tsf-form-title qa-audit-heading">Audit Log</h3>
      <div className="wh-table-card">
        <table className="wh-table">
          <thead>
            <tr>
              <th>Source</th>
              <th>Action</th>
              <th>ผล</th>
              <th>โดย</th>
              <th>เวลา</th>
            </tr>
          </thead>
          <tbody>
            {auditLog.map((log) => (
              <tr key={log.ID}>
                <td>{log.SourceTable}</td>
                <td>{log.Action}</td>
                <td>
                  <span
                    className={
                      log.ResultStatus === 'PASS' ? 'qa-result-badge qa-pass' : 'qa-result-badge qa-fail'
                    }
                  >
                    {log.ResultStatus}
                  </span>
                </td>
                <td>{log.Name}</td>
                <td>{new Date(log.ActionDatetime).toLocaleString('th-TH')}</td>
              </tr>
            ))}
            {auditLog.length === 0 && (
              <tr>
                <td colSpan={5} className="wh-empty-cell">
                  ยังไม่มีการยืนยันผล
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </>
  )
}