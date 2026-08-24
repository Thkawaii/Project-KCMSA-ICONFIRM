import { useCallback, useEffect, useMemo, useState } from 'react'
import { getImportLicenseAlerts } from '../api/importLicense.js'
import { getExportLicenseAlerts } from '../api/exportLicense.js'
import { useAppNavigate } from '../lib/nav.jsx'
import { formatThaiDate, daysLeftLabel } from '../lib/licenseExpiry.js'
import { wasShownThisWeek, markShownThisWeek } from '../lib/weeklyPopupDismiss.js'
import {
  ShieldCheckIcon,
  XMarkIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
  ChevronRightIcon,
} from './icons.jsx'
import './LicenseWeeklyPopup.css'

// ─────────────────────────────────────────────────────────────────────────────
// LicenseWeeklyPopup — ป๊อปอัปสรุป "สถานะอายุใบอนุญาต" (นำเข้า + ส่งออก)
//
// พฤติกรรม:
//   • เด้งขึ้นเอง "สัปดาห์ละครั้ง" เมื่อผู้ใช้ (role LOG) เข้าระบบครั้งแรกของสัปดาห์
//     เข้าวันไหนก็ได้ในสัปดาห์นั้น จะเห็นแค่ครั้งเดียว (จำผ่าน weeklyPopupDismiss)
//   • เด้งเฉพาะเมื่อมีใบ "หมดอายุ/ใกล้หมดอายุ" จริง ๆ — ถ้าไม่มีก็ไม่รบกวน
//     และยัง "ไม่เผา" สิทธิ์ของสัปดาห์นั้น (ค่อยเด้งเมื่อมีรายการโผล่ในสัปดาห์เดียวกัน)
//   • ไม่มีปุ่ม "เตือนอีกทีสัปดาห์หน้า" — เพราะระบบเตือนรายสัปดาห์ให้อยู่แล้ว
//     มีแค่ปุ่มเดียว "ไปจัดการใบอนุญาต"
//
// การ์ด/สไตล์ยกเครื่องให้ smooth: หัวการ์ดไล่มินต์อ่อนบนพื้นขาว, มุมโค้ง 28px,
//   ไอคอนวงกลมลอยตัว, ชิปพาสเทล 3 ช่อง, การ์ดรายการ hover ลอยขึ้น, ปุ่มไล่เขียวมินต์
//   ทุก easing ใช้ cubic-bezier(0.22,1,0.36,1) ให้ลื่นเป็นธรรมชาติ
// ─────────────────────────────────────────────────────────────────────────────

const CLOSE_MS = 320 // ให้ตรงกับ transition ตอนปิดใน CSS
const MAX_ROWS = 3 // แสดงรายการด่วนสุดกี่แถวในป๊อปอัป (ที่เหลือชี้ไปดูที่กระดิ่ง)

const isAlert = (i) => i.Status === 'EXPIRED' || i.Status === 'EXPIRING'

export default function LicenseWeeklyPopup() {
  const navigate = useAppNavigate()
  const [items, setItems] = useState([])
  const [open, setOpen] = useState(false)
  const [closing, setClosing] = useState(false)

  // โหลดข้อมูลรวมสองฝั่ง แล้วตัดสินใจว่าจะเด้งป๊อปอัปไหม (ครั้งเดียว ตอน mount)
  useEffect(() => {
    let alive = true
    async function load() {
      // ถ้าสัปดาห์นี้เคยเห็นแล้ว ไม่ต้องยิง API ให้เปลืองเลย
      if (wasShownThisWeek()) return

      const [imp, exp] = await Promise.allSettled([
        getImportLicenseAlerts({ onlyAlert: true }),
        getExportLicenseAlerts({ onlyAlert: true }),
      ])
      if (!alive) return

      const impList = (imp.status === 'fulfilled' ? imp.value?.items || [] : [])
        .filter(isAlert)
        .map((it) => ({ ...it, kind: 'import' }))
      const expList = (exp.status === 'fulfilled' ? exp.value?.items || [] : [])
        .filter(isAlert)
        .map((it) => ({ ...it, kind: 'export' }))

      const merged = [...impList, ...expList]
      // ไม่มีอะไรต้องเตือน -> ไม่เด้ง และไม่เผาสิทธิ์ของสัปดาห์นี้
      if (merged.length === 0) return

      // กันเด้งซ้ำ (เช่น StrictMode/สลับหน้า): เช็ค + ปั๊มความจำแบบ synchronous
      if (wasShownThisWeek()) return
      markShownThisWeek()

      setItems(merged)
      setOpen(true)
    }
    load()
    return () => {
      alive = false
    }
  }, [])

  // ปิดแบบนุ่ม: เล่นแอนิเมชันออกก่อน แล้วค่อยถอดออกจริง
  const handleClose = useCallback(() => {
    setClosing(true)
    const id = setTimeout(() => {
      setOpen(false)
      setClosing(false)
    }, CLOSE_MS)
    return () => clearTimeout(id)
  }, [])

  // ปิดด้วยปุ่ม Esc
  useEffect(() => {
    if (!open) return
    const onKey = (e) => {
      if (e.key === 'Escape') handleClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, handleClose])

  const { expired, expiring, total, rows } = useMemo(() => {
    const exp = items.filter((i) => i.Status === 'EXPIRED')
    const near = items.filter((i) => i.Status === 'EXPIRING')
    // เรียงด่วนสุดก่อน: DaysLeft น้อย/ติดลบมากขึ้นมาก่อน
    const sorted = [...items].sort((a, b) => (a.DaysLeft ?? 0) - (b.DaysLeft ?? 0))
    return {
      expired: exp.length,
      expiring: near.length,
      total: items.length,
      rows: sorted.slice(0, MAX_ROWS),
    }
  }, [items])

  if (!open) return null

  const titleOf = (it) =>
    it.kind === 'import' ? it.LicenseNo || 'ไม่มีเลขใบอนุญาต' : it.SerialNumber || '—'
  const metaOf = (it) =>
    it.kind === 'import'
      ? `นำเข้า · Invoice ${it.InvoiceNo || '—'}${it.Model ? ` · ${it.Model}` : ''}`
      : `ส่งออก · Exception ${it.ExceptionLicense || '—'}`

  // คลิกรายการ -> เปิดหน้าใบอนุญาตของฝั่งนั้น พร้อม auto-focus แถวที่คลิก
  const openItem = (it) => {
    handleClose()
    if (it.kind === 'import') {
      navigate('/warehouse', {
        focusLicense: it.LicenseNo || '',
        focusInvoice: it.InvoiceNo || '',
        focusTs: Date.now(),
      })
    } else {
      navigate('/warehouse/export-license', {
        focusSerial: it.SerialNumber || '',
        focusException: it.ExceptionLicense || '',
        focusTs: Date.now(),
      })
    }
  }

  const goManage = () => {
    handleClose()
    navigate('/warehouse')
  }

  return (
    <div
      className={'lwp-overlay' + (closing ? ' is-closing' : '')}
      role="presentation"
      onClick={handleClose}
    >
      <div
        className={'lwp-card' + (closing ? ' is-closing' : '')}
        role="dialog"
        aria-modal="true"
        aria-labelledby="lwp-title"
        onClick={(e) => e.stopPropagation()}
      >
        {/* ── หัวการ์ด: ไล่มินต์อ่อนบนพื้นขาว ── */}
        <div className="lwp-head">
          <button className="lwp-close" onClick={handleClose} aria-label="ปิด">
            <XMarkIcon className="size-4" />
          </button>

          <div className="lwp-badge-wrap">
            <span className="lwp-badge">
              <ShieldCheckIcon className="size-7" />
            </span>
          </div>

          <p className="lwp-eyebrow">แจ้งเตือนประจำสัปดาห์</p>
          <h2 className="lwp-title" id="lwp-title">
            สถานะอายุใบอนุญาต
          </h2>

          {/* ── ชิปสรุป 3 ช่อง ── */}
          <div className="lwp-chips">
            <div className="lwp-chip lwp-chip-expired">
              <span className="lwp-chip-num">{expired}</span>
              <span className="lwp-chip-lbl">หมดอายุแล้ว</span>
            </div>
            <div className="lwp-chip lwp-chip-expiring">
              <span className="lwp-chip-num">{expiring}</span>
              <span className="lwp-chip-lbl">ใกล้หมดอายุ</span>
            </div>
            <div className="lwp-chip lwp-chip-total">
              <span className="lwp-chip-num">{total}</span>
              <span className="lwp-chip-lbl">รวมทั้งหมด</span>
            </div>
          </div>
        </div>

        {/* ── รายการที่ต้องดำเนินการ ── */}
        <div className="lwp-body">
          <p className="lwp-section-label">รายการที่ต้องดำเนินการ</p>

          <div className="lwp-list">
            {rows.map((it, idx) => {
              const isExp = it.Status === 'EXPIRED'
              return (
                <button
                  key={`${it.kind}-${titleOf(it)}-${idx}`}
                  type="button"
                  className="lwp-item"
                  onClick={() => openItem(it)}
                  title="ดูในหน้าจัดการใบอนุญาต"
                >
                  <span className={'lwp-item-icon lwp-item-icon-' + it.kind}>
                    {it.kind === 'import' ? (
                      <ArrowDownTrayIcon className="size-4" />
                    ) : (
                      <ArrowUpTrayIcon className="size-4" />
                    )}
                  </span>

                  <span className="lwp-item-main">
                    <span className="lwp-item-title">{titleOf(it)}</span>
                    <span className="lwp-item-meta">{metaOf(it)}</span>
                  </span>

                  <span className="lwp-item-right">
                    <span
                      className={
                        'lwp-item-days ' + (isExp ? 'is-expired' : 'is-expiring')
                      }
                    >
                      {daysLeftLabel(it.DaysLeft)}
                    </span>
                    <span className="lwp-item-exp">หมดอายุ {formatThaiDate(it.ExpiryDate)}</span>
                  </span>
                </button>
              )
            })}
          </div>

          {total > rows.length && (
            <p className="lwp-more">
              และอีก {total - rows.length} รายการ ดูทั้งหมดได้ที่กระดิ่งแจ้งเตือน
            </p>
          )}
        </div>

        {/* ── ปุ่มหลักปุ่มเดียว ── */}
        <div className="lwp-foot">
          <button type="button" className="lwp-cta" onClick={goManage}>
            <span>ไปจัดการใบอนุญาต</span>
            <ChevronRightIcon className="size-4" />
          </button>
          <p className="lwp-brand">ระบบแจ้งเตือนอัตโนมัติ · KOBELCO I-CONFIRMATION</p>
        </div>
      </div>
    </div>
  )
}
