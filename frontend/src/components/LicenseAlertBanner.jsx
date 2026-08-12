import { useEffect, useState } from 'react'
import { getImportLicenseAlerts } from '../api/importLicense.js'
import { getExportLicenseAlerts } from '../api/exportLicense.js'
import { formatThaiDate, daysLeftLabel } from '../lib/licenseExpiry.js'
import { ExclamationTriangleIcon, ClockIcon, CheckCircleIcon } from './icons.jsx'

// ─────────────────────────────────────────────────────────────────────────────
// LicenseAlertBanner — แถบสรุปสถานะอายุใบอนุญาต "แบบถาวร" บนหน้า License
//
// ต่างจากกระดิ่ง (ที่กดอ่านแล้วป้าย "ใหม่" หาย) — แถบนี้ผูกกับข้อมูลจริงใน DB
// จึงแสดงอยู่ตลอดตราบใดที่ยังมีใบใกล้หมด/หมดอายุ ไม่ว่าจะกดอ่านหรือยัง
// ทำให้ผู้ใช้ "รับรู้สถานะจริง" ตลอดเวลา ไม่ใช่แค่ตอนเปิดกระดิ่งครั้งแรก
//
// props:
//   kind = 'import' | 'export'
//   onOpenItem(item) — (ไม่บังคับ) คลิกรายการเพื่อไฮไลต์แถวในตารางด้านล่าง
// ─────────────────────────────────────────────────────────────────────────────

const POLL_MS = 60_000

export default function LicenseAlertBanner({ kind = 'import', onOpenItem }) {
  const [items, setItems] = useState([])
  const [loaded, setLoaded] = useState(false)

  const isImport = kind === 'import'
  const fetcher = isImport ? getImportLicenseAlerts : getExportLicenseAlerts
  const windowLabel = isImport ? 'ภายใน 30 วัน' : 'ภายใน 7 วัน'

  useEffect(() => {
    let alive = true
    async function load() {
      try {
        const res = await fetcher({ onlyAlert: true })
        if (alive) {
          setItems(res?.items || [])
          setLoaded(true)
        }
      } catch {
        if (alive) setLoaded(true)
      }
    }
    load()
    const id = setInterval(load, POLL_MS)
    const onFocus = () => load()
    window.addEventListener('focus', onFocus)
    return () => {
      alive = false
      clearInterval(id)
      window.removeEventListener('focus', onFocus)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind])

  if (!loaded) return null

  const expired = items.filter((i) => i.Status === 'EXPIRED')
  const expiring = items.filter((i) => i.Status === 'EXPIRING')
  const total = expired.length + expiring.length

  const titleOf = (it) => (isImport ? it.LicenseNo || 'ไม่มีเลขใบอนุญาต' : it.SerialNumber || '—')
  const metaOf = (it) =>
    isImport
      ? `Invoice ${it.InvoiceNo || '—'}${it.Model ? ` · ${it.Model}` : ''}`
      : `Exception ${it.ExceptionLicense || '—'}`

  // ไม่มีอะไรใกล้หมด/หมดอายุ → แถบเขียวยืนยันความเรียบร้อย (ยังคงแสดงให้เห็นสถานะ)
  if (total === 0) {
    return (
      <div className="lab-banner lab-banner-ok" role="status">
        <span className="lab-banner-icon">
          <CheckCircleIcon className="size-5" />
        </span>
        <div className="lab-banner-body">
          <strong>ใบอนุญาต{isImport ? 'นำเข้า' : 'ส่งออก'}ทั้งหมดอยู่ในอายุ</strong>
          <span className="lab-banner-sub">ไม่มีรายการใกล้หมดอายุ ({windowLabel}) หรือหมดอายุ</span>
        </div>
      </div>
    )
  }

  const tone = expired.length > 0 ? 'danger' : 'warn'
  // เรียงตามความเร่งด่วน: หมดอายุก่อน แล้ว DaysLeft น้อยสุดก่อน
  const urgent = [...expired, ...expiring]
    .sort((a, b) => (a.DaysLeft ?? 0) - (b.DaysLeft ?? 0))
    .slice(0, 4)

  return (
    <div className={`lab-banner lab-banner-${tone}`} role="alert">
      <span className="lab-banner-icon">
        {expired.length > 0 ? (
          <ExclamationTriangleIcon className="size-5" />
        ) : (
          <ClockIcon className="size-5" />
        )}
      </span>
      <div className="lab-banner-body">
        <div className="lab-banner-head">
          <strong>
            ใบอนุญาต{isImport ? 'นำเข้า' : 'ส่งออก'}ต้องดำเนินการ {total} รายการ
          </strong>
          <div className="lab-banner-chips">
            {expired.length > 0 && (
              <span className="lab-banner-chip lab-banner-chip-expired">
                หมดอายุแล้ว {expired.length}
              </span>
            )}
            {expiring.length > 0 && (
              <span className="lab-banner-chip lab-banner-chip-expiring">
                ใกล้หมดอายุ {expiring.length}
              </span>
            )}
          </div>
        </div>

        <div className="lab-banner-list">
          {urgent.map((it, idx) => {
            const isExp = it.Status === 'EXPIRED'
            return (
              <button
                key={`${titleOf(it)}-${idx}`}
                type="button"
                className="lab-banner-item"
                onClick={() => onOpenItem?.(it)}
                title={onOpenItem ? 'ดูในตารางด้านล่าง' : undefined}
              >
                <span className={`lab-banner-dot ${isExp ? 'is-expired' : 'is-expiring'}`} />
                <span className="lab-banner-item-title">{titleOf(it)}</span>
                <span className="lab-banner-item-meta">{metaOf(it)}</span>
                <span className={`lab-banner-item-days ${isExp ? 'is-expired' : 'is-expiring'}`}>
                  {daysLeftLabel(it.DaysLeft)} · หมดอายุ {formatThaiDate(it.ExpiryDate)}
                </span>
              </button>
            )
          })}
          {total > urgent.length && (
            <span className="lab-banner-more">และอีก {total - urgent.length} รายการในตารางด้านล่าง</span>
          )}
        </div>
      </div>
    </div>
  )
}
