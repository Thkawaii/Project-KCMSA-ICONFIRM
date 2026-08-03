import { useCallback, useEffect, useRef, useState } from 'react'
import { getImportLicenseAlerts } from '../api/importLicense.js'
import { useAppNavigate } from '../lib/nav.jsx'
import { formatThaiDate, daysLeftLabel } from '../lib/licenseExpiry.js'
import { BellAlertIcon, XMarkIcon, DocumentTextIcon, ClockIcon } from './icons.jsx'

// ─────────────────────────────────────────────────────────────────────────────
// LicenseAlertBell — กระดิ่งแจ้งเตือนอายุใบอนุญาตนำเข้าบน topbar
//
// • ดึงรายการที่ "หมดอายุ / ใกล้หมดอายุ" ทุก 60 วิ (+ ตอนสลับกลับมาที่แท็บ) = realtime
// • badge สีแดง = จำนวนใบที่ต้องรีบจัดการ  ·  วงกระเพื่อม (pulse) เมื่อมีของใหม่
//   ตั้งแต่ครั้งล่าสุดที่เปิดดู (จำผ่าน localStorage) — ให้ความรู้สึก "เตือนรายสัปดาห์"
// • คลิก -> panel จัดกลุ่ม หมดอายุแล้ว / ใกล้หมดอายุ เรียงด่วนสุดขึ้นก่อน
//
// แสดงเฉพาะ role WH (คนเดียวที่มีสิทธิ์ /import-license) — role อื่นจะไม่เห็นกระดิ่ง
// และไม่ยิง API ที่จะโดน 403
// ─────────────────────────────────────────────────────────────────────────────

const POLL_MS = 60_000
const ACK_KEY = 'iconfirm_license_alert_ack' // จำนวน alert ที่ผู้ใช้เห็นล่าสุด

export default function LicenseAlertBell() {
  const navigate = useAppNavigate()
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState([])
  const [counts, setCounts] = useState({ alert: 0, expired: 0, expiring: 0, noDate: 0 })
  const [loaded, setLoaded] = useState(false)
  const [hasNew, setHasNew] = useState(false)
  const rootRef = useRef(null)

  const load = useCallback(async () => {
    try {
      const data = await getImportLicenseAlerts({ onlyAlert: true })
      const c = data?.counts || {}
      setItems(data?.items || [])
      setCounts({
        alert: c.alert || 0,
        expired: c.expired || 0,
        expiring: c.expiring || 0,
        noDate: c.noDate || 0,
      })
      // เทียบกับจำนวนที่ผู้ใช้เห็นล่าสุด -> มีของใหม่ไหม
      const ack = Number(localStorage.getItem(ACK_KEY) || 0)
      setHasNew((c.alert || 0) > ack)
      setLoaded(true)
    } catch {
      // เงียบไว้ — กระดิ่งพังไม่ควรทำทั้งหน้าล้ม (เช่น token หมดอายุ)
      setLoaded(true)
    }
  }, [])

  useEffect(() => {
    load()
    const id = setInterval(load, POLL_MS)
    const onFocus = () => load()
    window.addEventListener('focus', onFocus)
    return () => {
      clearInterval(id)
      window.removeEventListener('focus', onFocus)
    }
  }, [load])

  // ปิด panel เมื่อคลิกนอกกล่อง / กด Esc
  useEffect(() => {
    if (!open) return
    function onDown(e) {
      if (rootRef.current && !rootRef.current.contains(e.target)) setOpen(false)
    }
    function onKey(e) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  function toggle() {
    const next = !open
    setOpen(next)
    if (next) {
      // เปิดดู = รับรู้แล้ว หยุดกระเพื่อม และจำจำนวนล่าสุดไว้
      localStorage.setItem(ACK_KEY, String(counts.alert))
      setHasNew(false)
    }
  }

  function goToLicense() {
    setOpen(false)
    navigate('/warehouse')
  }

  const expired = items.filter((it) => it.Status === 'EXPIRED')
  const expiring = items.filter((it) => it.Status === 'EXPIRING')
  const badge = counts.alert

  return (
    <div className="lab-root" ref={rootRef}>
      <button
        type="button"
        className={'lab-bell' + (badge > 0 ? ' lab-bell-active' : '') + (hasNew ? ' lab-bell-pulse' : '')}
        onClick={toggle}
        aria-label={`แจ้งเตือนอายุใบอนุญาต${badge > 0 ? ` (${badge})` : ''}`}
        title="แจ้งเตือนอายุใบอนุญาตนำเข้า"
      >
        <span className="lab-bell-icon">
          <BellAlertIcon className="size-5" />
        </span>
        {badge > 0 && <span className="lab-badge">{badge > 99 ? '99+' : badge}</span>}
      </button>

      {open && (
        <div className="lab-panel" role="dialog" aria-label="แจ้งเตือนอายุใบอนุญาต">
          <div className="lab-panel-head">
            <div>
              <h3 className="lab-panel-title">อายุใบอนุญาตนำเข้า</h3>
              <p className="lab-panel-sub">ใบอนุญาตมีอายุ 6 เดือน · ตรวจสอบรายสัปดาห์</p>
            </div>
            <button className="lab-panel-close" onClick={() => setOpen(false)} aria-label="ปิด">
              <XMarkIcon className="size-4" />
            </button>
          </div>

          {/* แถบสรุปตัวเลข */}
          <div className="lab-summary">
            <div className="lab-sum-chip lab-sum-expired">
              <span className="lab-sum-num">{counts.expired}</span>
              <span className="lab-sum-lbl">หมดอายุแล้ว</span>
            </div>
            <div className="lab-sum-chip lab-sum-expiring">
              <span className="lab-sum-num">{counts.expiring}</span>
              <span className="lab-sum-lbl">ใกล้หมดอายุ</span>
            </div>
          </div>

          <div className="lab-list">
            {!loaded && <div className="lab-empty">กำลังโหลด...</div>}

            {loaded && badge === 0 && (
              <div className="lab-empty lab-empty-ok">
                <span className="lab-empty-dot" />
                ทุกใบอนุญาตยังอยู่ในอายุ ไม่มีรายการต้องจัดการ
              </div>
            )}

            {expired.length > 0 && (
              <>
                <div className="lab-group-label lab-group-expired">หมดอายุแล้ว</div>
                {expired.map((it, i) => (
                  <AlertItem key={`e${i}`} item={it} onClick={goToLicense} />
                ))}
              </>
            )}

            {expiring.length > 0 && (
              <>
                <div className="lab-group-label lab-group-expiring">ใกล้หมดอายุ (ภายใน 30 วัน)</div>
                {expiring.map((it, i) => (
                  <AlertItem key={`s${i}`} item={it} onClick={goToLicense} />
                ))}
              </>
            )}
          </div>

          {counts.noDate > 0 && (
            <div className="lab-foot-note">
              <ClockIcon className="size-4" />
              มี {counts.noDate} ใบที่ยังไม่ได้ระบุวันที่ออกใบอนุญาต — เติมวันที่เพื่อให้ระบบเตือนอายุได้
            </div>
          )}

          <button className="lab-foot-link" onClick={goToLicense}>
            <DocumentTextIcon className="size-4" />
            เปิดตาราง Import License
          </button>
        </div>
      )}
    </div>
  )
}

function AlertItem({ item, onClick }) {
  const isExpired = item.Status === 'EXPIRED'
  return (
    <button className="lab-item" onClick={onClick}>
      <span className={'lab-item-bar ' + (isExpired ? 'lab-bar-expired' : 'lab-bar-expiring')} />
      <span className="lab-item-body">
        <span className="lab-item-top">
          <span className="lab-item-license">{item.LicenseNo || '—'}</span>
          <span className={'lab-item-days ' + (isExpired ? 'lab-days-expired' : 'lab-days-expiring')}>
            {daysLeftLabel(item.DaysLeft)}
          </span>
        </span>
        <span className="lab-item-meta">
          Invoice {item.InvoiceNo || '—'}
          {item.Model ? ` · ${item.Model}` : ''} · {item.Total} เครื่อง
        </span>
        <span className="lab-item-expiry">หมดอายุ {formatThaiDate(item.ExpiryDate)}</span>
      </span>
    </button>
  )
}
