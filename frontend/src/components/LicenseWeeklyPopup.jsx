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


const CLOSE_MS = 320
const MAX_ROWS = 3

const isAlert = (i) => i.Status === 'EXPIRED' || i.Status === 'EXPIRING'

export default function LicenseWeeklyPopup() {
  const navigate = useAppNavigate()
  const [items, setItems] = useState([])
  const [open, setOpen] = useState(false)
  const [closing, setClosing] = useState(false)

  useEffect(() => {
    let alive = true
    async function load() {
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
      if (merged.length === 0) return

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

  const handleClose = useCallback(() => {
    setClosing(true)
    const id = setTimeout(() => {
      setOpen(false)
      setClosing(false)
    }, CLOSE_MS)
    return () => clearTimeout(id)
  }, [])

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
