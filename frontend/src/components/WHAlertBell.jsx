import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { getImportLicenseAlerts } from '../api/importLicense.js'
import { getExportLicenseAlerts } from '../api/exportLicense.js'
import { useAppNavigate } from '../lib/nav.jsx'
import { formatThaiDate, daysLeftLabel } from '../lib/licenseExpiry.js'
import {
  addDismissed,
  clearDismissed,
  dismissKey,
  pruneDismissed,
  readDismissed,
  removeDismissed,
} from '../lib/licenseDismiss.js'
import {
  addExportDismissed,
  clearExportDismissed,
  exportDismissKey,
  pruneExportDismissed,
  readExportDismissed,
  removeExportDismissed,
} from '../lib/exportLicenseDismiss.js'
import {
  BellAlertIcon,
  XMarkIcon,
  ClockIcon,
  EyeSlashIcon,
  ArrowPathIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
} from './icons.jsx'

// ─────────────────────────────────────────────────────────────────────────────
// WHAlertBell — กระดิ่งแจ้งเตือนอายุใบอนุญาต "รวมนำเข้า + ส่งออก" ไว้ในอันเดียว
//
// เดิมมี 2 กระดิ่งแยกกัน (นำเข้า/ส่งออก) วางข้างกันบน topbar — รวมเป็นอันเดียว:
//   • badge = จำนวนที่ต้องจัดการรวมทั้งสองฝั่ง (ที่ยังไม่ถูกซ่อน)
//   • เปิด panel เดียว แบ่งเป็น 2 ส่วน: นำเข้า (6 เดือน) / ส่งออก (1 เดือน)
//   • ซ่อน/คืนค่ารายการได้ทั้งสองฝั่ง (ใช้คลัง dismiss เดิมของแต่ละฝั่ง)
//   • คลิกรายการ → เด้งไปหน้าที่เกี่ยวข้อง (Import / Export License)
//
// แสดงเฉพาะ WH Manager (คนเดียวที่เข้าถึงบัญชีใบอนุญาต) — role อื่นไม่เห็นและไม่ยิง API
// ─────────────────────────────────────────────────────────────────────────────

const POLL_MS = 60_000
const ACK_KEY = 'iconfirm_wh_alert_ack' // จำนวน alert รวมที่ผู้ใช้เห็นล่าสุด (เฉพาะที่ยังไม่ซ่อน)

export default function WHAlertBell() {
  const navigate = useAppNavigate()
  const [open, setOpen] = useState(false)

  const [impItems, setImpItems] = useState([])
  const [expItems, setExpItems] = useState([])
  const [impNoDate, setImpNoDate] = useState(0)
  const [expNoDate, setExpNoDate] = useState(0)
  const [loaded, setLoaded] = useState(false)
  const [hasNew, setHasNew] = useState(false)

  const [impDismissed, setImpDismissed] = useState(() => readDismissed())
  const [expDismissed, setExpDismissed] = useState(() => readExportDismissed())
  const [showHiddenImp, setShowHiddenImp] = useState(false)
  const [showHiddenExp, setShowHiddenExp] = useState(false)
  const rootRef = useRef(null)

  const load = useCallback(async () => {
    // ดึงทั้งสองฝั่งพร้อมกัน — ฝั่งไหนพังก็ไม่ล้มอีกฝั่ง (กระดิ่งพังไม่ควรทำทั้งหน้าล้ม)
    const [imp, exp] = await Promise.allSettled([
      getImportLicenseAlerts({ onlyAlert: true }),
      getExportLicenseAlerts({ onlyAlert: true }),
    ])
    if (imp.status === 'fulfilled') {
      const list = imp.value?.items || []
      setImpItems(list)
      setImpNoDate(imp.value?.counts?.noDate || 0)
      setImpDismissed(pruneDismissed(list))
    }
    if (exp.status === 'fulfilled') {
      const list = exp.value?.items || []
      setExpItems(list)
      setExpNoDate(exp.value?.counts?.noDate || 0)
      setExpDismissed(pruneExportDismissed(list))
    }
    setLoaded(true)
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

  // แยก visible/hidden ของแต่ละฝั่ง
  const imp = useMemo(() => {
    const isHidden = (it) => Object.prototype.hasOwnProperty.call(impDismissed, dismissKey(it))
    const vis = impItems.filter((it) => !isHidden(it))
    return {
      hidden: impItems.filter(isHidden),
      expired: vis.filter((it) => it.Status === 'EXPIRED'),
      expiring: vis.filter((it) => it.Status === 'EXPIRING'),
      visible: vis.length,
    }
  }, [impItems, impDismissed])

  const exp = useMemo(() => {
    const isHidden = (it) => Object.prototype.hasOwnProperty.call(expDismissed, exportDismissKey(it))
    const vis = expItems.filter((it) => !isHidden(it))
    return {
      hidden: expItems.filter(isHidden),
      expired: vis.filter((it) => it.Status === 'EXPIRED'),
      expiring: vis.filter((it) => it.Status === 'EXPIRING'),
      visible: vis.length,
    }
  }, [expItems, expDismissed])

  const totalVisible = imp.visible + exp.visible

  useEffect(() => {
    if (!loaded) return
    const ack = Number(localStorage.getItem(ACK_KEY) || 0)
    setHasNew(totalVisible > ack)
  }, [loaded, totalVisible])

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
      localStorage.setItem(ACK_KEY, String(totalVisible))
      setHasNew(false)
    } else {
      setShowHiddenImp(false)
      setShowHiddenExp(false)
    }
  }

  // ── import handlers ──
  const goImport = () => {
    setOpen(false)
    navigate('/warehouse')
  }
  const dismissImp = (item) => {
    setImpDismissed({ ...addDismissed(item) })
    localStorage.setItem(ACK_KEY, String(Math.max(0, totalVisible - 1)))
  }
  const restoreImp = (item) => setImpDismissed({ ...removeDismissed(item) })

  // ── export handlers ──
  const goExport = () => {
    setOpen(false)
    navigate('/warehouse/export-license')
  }
  const dismissExp = (item) => {
    setExpDismissed({ ...addExportDismissed(item) })
    localStorage.setItem(ACK_KEY, String(Math.max(0, totalVisible - 1)))
  }
  const restoreExp = (item) => setExpDismissed({ ...removeExportDismissed(item) })

  function restoreAll() {
    setImpDismissed(clearDismissed())
    setExpDismissed(clearExportDismissed())
    setShowHiddenImp(false)
    setShowHiddenExp(false)
  }

  const badge = totalVisible
  const totalHidden = imp.hidden.length + exp.hidden.length

  return (
    <div className="lab-root" ref={rootRef}>
      <button
        type="button"
        className={'lab-bell' + (badge > 0 ? ' lab-bell-active' : '') + (hasNew ? ' lab-bell-pulse' : '')}
        onClick={toggle}
        aria-label={`แจ้งเตือนอายุใบอนุญาต${badge > 0 ? ` (${badge})` : ''}`}
        title="แจ้งเตือนอายุใบอนุญาต (นำเข้า + ส่งออก)"
      >
        <span className="lab-bell-icon">
          <BellAlertIcon className="size-5" />
        </span>
        {badge > 0 && <span className="lab-badge">{badge > 99 ? '99+' : badge}</span>}
      </button>

      {open && (
        <div className="lab-panel lab-panel-wide" role="dialog" aria-label="แจ้งเตือนอายุใบอนุญาต">
          <div className="lab-panel-head">
            <div>
              <h3 className="lab-panel-title">แจ้งเตือนอายุใบอนุญาต</h3>
              <p className="lab-panel-sub">รวมนำเข้า (6 เดือน) และส่งออก (1 เดือน) · ตรวจสอบรายสัปดาห์</p>
            </div>
            <button className="lab-panel-close" onClick={() => setOpen(false)} aria-label="ปิด">
              <XMarkIcon className="size-4" />
            </button>
          </div>

          {loaded && badge === 0 && totalHidden === 0 && (
            <div className="lab-empty lab-empty-ok" style={{ margin: 12 }}>
              <span className="lab-empty-dot" />
              ทุกใบอนุญาต (นำเข้า/ส่งออก) ยังอยู่ในอายุ ไม่มีรายการต้องจัดการ
            </div>
          )}

          {/* ── ส่วนนำเข้า ── */}
          <AlertSection
            theme="import"
            title="ใบอนุญาตนำเข้า"
            kindIcon={<ArrowDownTrayIcon className="size-3" />}
            loaded={loaded}
            expired={imp.expired}
            expiring={imp.expiring}
            hidden={imp.hidden}
            showHidden={showHiddenImp}
            onToggleHidden={() => setShowHiddenImp((v) => !v)}
            noDate={impNoDate}
            noDateLabel="ยังไม่ได้ระบุวันที่ออกใบอนุญาต"
            expiringLabel="ใกล้หมดอายุ (ภายใน 30 วัน)"
            onOpen={goImport}
            onDismiss={dismissImp}
            onRestore={restoreImp}
            getKey={dismissKey}
            renderMeta={(it) => (
              <>
                Invoice {it.InvoiceNo || '—'}
                {it.Model ? ` · ${it.Model}` : ''} · {it.Total} เครื่อง
              </>
            )}
            titleField={(it) => it.LicenseNo || '—'}
          />

          {/* ── ส่วนส่งออก ── */}
          <AlertSection
            theme="export"
            title="ใบอนุญาตส่งออก"
            kindIcon={<ArrowUpTrayIcon className="size-3" />}
            loaded={loaded}
            expired={exp.expired}
            expiring={exp.expiring}
            hidden={exp.hidden}
            showHidden={showHiddenExp}
            onToggleHidden={() => setShowHiddenExp((v) => !v)}
            noDate={expNoDate}
            noDateLabel="ยังไม่ได้ระบุวันหมดอายุ/ใบขน"
            expiringLabel="ใกล้หมดอายุ (ภายใน 7 วัน)"
            onOpen={goExport}
            onDismiss={dismissExp}
            onRestore={restoreExp}
            getKey={exportDismissKey}
            renderMeta={(it) => (
              <>
                Exception License {it.ExceptionLicense || '—'}
                {it.DeclarationDate ? ` · ใบขน ${formatThaiDate(it.DeclarationDate)}` : ''}
              </>
            )}
            titleField={(it) => it.SerialNumber || '—'}
          />

          {totalHidden > 0 && (
            <div className="lab-hidden-bar">
              <span style={{ fontSize: 12, color: '#94a3b8' }}>ซ่อนไว้ {totalHidden} รายการ</span>
              <button className="lab-hidden-restore" onClick={restoreAll} title="คืนค่าการแจ้งเตือนทั้งหมด">
                <ArrowPathIcon className="size-4" />
                คืนค่าทั้งหมด
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function AlertSection({
  theme,
  title,
  kindIcon,
  loaded,
  expired,
  expiring,
  hidden,
  showHidden,
  onToggleHidden,
  noDate,
  noDateLabel,
  expiringLabel,
  onOpen,
  onDismiss,
  onRestore,
  getKey,
  renderMeta,
  titleField,
}) {
  const visible = expired.length + expiring.length
  const themeClass = theme === 'export' ? ' lab-section-export' : ' lab-section-import'

  return (
    <div className={'lab-section' + themeClass}>
      <div className="lab-section-head">
        <span className={'lab-section-kind lab-kind-' + theme}>{kindIcon}</span>
        <span className="lab-section-title">{title}</span>
        <span className="lab-section-count">
          <span className="lab-sum-chip lab-sum-expired">
            <span className="lab-sum-num">{expired.length}</span>
            <span className="lab-sum-lbl">หมดอายุ</span>
          </span>
          <span className="lab-sum-chip lab-sum-expiring">
            <span className="lab-sum-num">{expiring.length}</span>
            <span className="lab-sum-lbl">ใกล้หมด</span>
          </span>
        </span>
      </div>

      <div className="lab-list">
        {loaded && visible === 0 && (
          <div className="lab-empty lab-empty-ok">
            <span className="lab-empty-dot" />
            ไม่มีรายการต้องจัดการ
          </div>
        )}

        {expired.length > 0 && (
          <>
            <div className="lab-group-label lab-group-expired">หมดอายุแล้ว</div>
            {expired.map((it) => (
              <Row key={getKey(it)} item={it} onOpen={onOpen} onDismiss={onDismiss} renderMeta={renderMeta} titleField={titleField} />
            ))}
          </>
        )}

        {expiring.length > 0 && (
          <>
            <div className="lab-group-label lab-group-expiring">{expiringLabel}</div>
            {expiring.map((it) => (
              <Row key={getKey(it)} item={it} onOpen={onOpen} onDismiss={onDismiss} renderMeta={renderMeta} titleField={titleField} />
            ))}
          </>
        )}

        {showHidden && hidden.length > 0 && (
          <>
            <div className="lab-group-label lab-group-hidden">ซ่อนไว้</div>
            {hidden.map((it) => (
              <Row key={getKey(it)} item={it} hidden onOpen={onOpen} onRestore={onRestore} renderMeta={renderMeta} titleField={titleField} />
            ))}
          </>
        )}
      </div>

      {noDate > 0 && (
        <div className="lab-foot-note">
          <ClockIcon className="size-4" />
          มี {noDate} ใบที่{noDateLabel} — เติมวันที่เพื่อให้ระบบเตือนอายุได้
        </div>
      )}

      {hidden.length > 0 && (
        <button className="lab-hidden-toggle" onClick={onToggleHidden} style={{ margin: '2px 12px 10px' }}>
          <EyeSlashIcon className="size-4" />
          {showHidden ? 'ซ่อนรายการที่ซ่อนไว้' : `ดูที่ซ่อนไว้ ${hidden.length} รายการ`}
        </button>
      )}
    </div>
  )
}

function Row({ item, hidden = false, onOpen, onDismiss, onRestore, renderMeta, titleField }) {
  const isExpired = item.Status === 'EXPIRED'
  return (
    <div className={'lab-item' + (hidden ? ' lab-item-hidden' : '')}>
      <button className="lab-item-main" onClick={() => onOpen?.(item)}>
        <span className={'lab-item-bar ' + (isExpired ? 'lab-bar-expired' : 'lab-bar-expiring')} />
        <span className="lab-item-body">
          <span className="lab-item-top">
            <span className="lab-item-license">{titleField(item)}</span>
            <span className={'lab-item-days ' + (isExpired ? 'lab-days-expired' : 'lab-days-expiring')}>
              {daysLeftLabel(item.DaysLeft)}
            </span>
          </span>
          <span className="lab-item-meta">{renderMeta(item)}</span>
          <span className="lab-item-expiry">หมดอายุ {formatThaiDate(item.ExpiryDate)}</span>
        </span>
      </button>

      {hidden ? (
        <button
          type="button"
          className="lab-item-action lab-item-restore"
          onClick={() => onRestore?.(item)}
          aria-label="คืนค่าการแจ้งเตือนนี้"
          title="คืนค่าการแจ้งเตือน"
        >
          <ArrowPathIcon className="size-4" />
        </button>
      ) : (
        <button
          type="button"
          className="lab-item-action lab-item-dismiss"
          onClick={() => onDismiss?.(item)}
          aria-label="ซ่อนการแจ้งเตือนนี้"
          title="ซ่อนการแจ้งเตือน (ไม่ลบใบอนุญาต)"
        >
          <XMarkIcon className="size-4" />
        </button>
      )}
    </div>
  )
}
