import { useMemo } from 'react'
import { useOutletContext } from 'react-router-dom'
import { componentNameByPartNo } from './QALayout.jsx'

export default function QADashboard() {
  const { queue, auditLog } = useOutletContext()

  const total = queue.length
  const passed = queue.filter((q) => q.Result === 'PASS').length
  const failed = queue.filter((q) => q.Result === 'FAIL').length
  const passRate = total ? Math.round((passed / total) * 100) : 0

  // สร้างกราฟรายวันของเดือนปัจจุบัน จากข้อมูลใน auditLog จริง
  const trend = useMemo(() => {
    const now = new Date()
    const year = now.getFullYear()
    const month = now.getMonth()
    const daysInMonth = new Date(year, month + 1, 0).getDate()

    const days = Array.from({ length: daysInMonth }, (_, i) => ({
      day: i + 1,
      pass: 0,
      fail: 0,
    }))

    for (const log of auditLog) {
      const d = log.action_date
      if (d && d.getFullYear() === year && d.getMonth() === month) {
        const bucket = days[d.getDate() - 1]
        if (log.ResultStatus === 'PASS') bucket.pass += 1
        else if (log.ResultStatus === 'FAIL') bucket.fail += 1
      }
    }

    return { days, year, month }
  }, [auditLog])

  const maxCount = Math.max(1, ...trend.days.map((d) => Math.max(d.pass, d.fail)))

  const componentSummary = useMemo(() => {
    const groups = {}
    for (const row of queue) {
      const key = row.ActualPartNo
      if (!groups[key]) groups[key] = { part_no: key, total: 0, passed: 0 }
      groups[key].total += 1
      if (row.Result === 'PASS') groups[key].passed += 1
    }
    return Object.values(groups)
  }, [queue])

  const monthLabel = new Date(trend.year, trend.month, 1).toLocaleDateString('en-US', {
    month: 'long',
    year: 'numeric',
  })

  return (
    <>
      <div className="qa-page-heading">
        <h2 className="wh-title">Dashboard</h2>
        <p className="wh-subtitle">Overview of validation activities and production status</p>
      </div>

      <div className="dash-stats-row">
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>Total Scans</span>
            <span className="dash-stat-icon dash-icon-blue">▦</span>
          </div>
          <div className="dash-stat-value">{total}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>Pass Rate</span>
            <span className="dash-stat-icon dash-icon-green">✓</span>
          </div>
          <div className="dash-stat-value">{passRate}%</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>Failed</span>
            <span className="dash-stat-icon dash-icon-red">✕</span>
          </div>
          <div className="dash-stat-value">{failed}</div>
        </div>
        <div className="dash-stat-card">
          <div className="dash-stat-label">
            <span>Active Orders</span>
            <span className="dash-stat-icon dash-icon-yellow">📋</span>
          </div>
          <div className="dash-stat-value">1</div>
        </div>
      </div>

      <div className="dash-chart-card">
        <div className="dash-chart-header">
          <h3 className="tsf-form-title">📈 Daily Pass &amp; Fail Trends</h3>
          <span className="dash-chart-month">{monthLabel}</span>
        </div>

        <div className="dash-chart-area">
          {trend.days.map((d) => (
            <div className="dash-chart-col" key={d.day}>
              <div className="dash-chart-bars">
                {d.pass > 0 && (
                  <div
                    className="dash-bar dash-bar-pass"
                    style={{ height: `${(d.pass / maxCount) * 100}%` }}
                    title={`${d.day}: ${d.pass} pass`}
                  />
                )}
                {d.fail > 0 && (
                  <div
                    className="dash-bar dash-bar-fail"
                    style={{ height: `${(d.fail / maxCount) * 100}%` }}
                    title={`${d.day}: ${d.fail} fail`}
                  />
                )}
              </div>
              <span className="dash-chart-day-label">{d.day}</span>
            </div>
          ))}
        </div>

        <div className="dash-chart-legend">
          <span>
            <span className="dash-legend-dot dash-legend-pass" /> Pass
          </span>
          <span>
            <span className="dash-legend-dot dash-legend-fail" /> Fail
          </span>
        </div>
      </div>

      <div className="dash-bottom-row">
        <div className="dash-summary-card">
          <h3 className="tsf-form-title">Recent Validation Activity</h3>
          <table className="wh-table">
            <thead>
              <tr>
                <th>Action</th>
                <th>โดย</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {auditLog.slice(0, 6).map((log) => (
                <tr key={log.ID}>
                  <td>{log.Action}</td>
                  <td>{log.Name}</td>
                  <td>
                    <span
                      className={
                        log.ResultStatus === 'PASS' ? 'qa-result-badge qa-pass' : 'qa-result-badge qa-fail'
                      }
                    >
                      {log.ResultStatus}
                    </span>
                  </td>
                </tr>
              ))}
              {auditLog.length === 0 && (
                <tr>
                  <td colSpan={3} className="wh-empty-cell">
                    ยังไม่มีกิจกรรม
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="dash-summary-card">
          <h3 className="tsf-form-title">Component Validation Summary</h3>
          {componentSummary.map((c) => (
            <div className="dash-summary-row" key={c.part_no}>
              <div className="dash-summary-row-top">
                <span>{componentNameByPartNo[c.part_no] || c.part_no}</span>
                <span className="dash-summary-fraction">
                  {c.passed}/{c.total} passed
                </span>
              </div>
              <div className="dash-progress-track">
                <div
                  className="dash-progress-fill dash-progress-green"
                  style={{ width: `${c.total ? (c.passed / c.total) * 100 : 0}%` }}
                />
              </div>
            </div>
          ))}
          {componentSummary.length === 0 && <p className="wh-subtitle">ยังไม่มีข้อมูล</p>}
        </div>
      </div>
    </>
  )
}