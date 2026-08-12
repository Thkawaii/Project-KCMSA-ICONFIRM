import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { getImportLicenseAlerts } from '../api/importLicense.js'
import { getExportLicenseAlerts } from '../api/exportLicense.js'
import { useAppNavigate } from '../lib/nav.jsx'
import { formatThaiDate, daysLeftLabel } from '../lib/licenseExpiry.js'
import { dismissKey } from '../lib/licenseDismiss.js'
import { exportDismissKey } from '../lib/exportLicenseDismiss.js'
import {
  BellAlertIcon,
  XMarkIcon,
  ClockIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
} from './icons.jsx'

// ─────────────────────────────────────────────────────────────────────────────
// WHAlertBell — กระดิ่งแจ้งเตือนอายุใบอนุญาต "รวมนำเข้า + ส่งออก" ไว้ในอันเดียว
//
// พฤติกรรมตัวเลข badge (ปรับใหม่):
//   • badge = จำนวนแจ้งเตือน "ที่ยังไม่ได้เปิดดู" (unseen) รวมสองฝั่ง
//   • พอกดเปิดกระดิ่ง = รับรู้แล้ว -> badge หายเป็น 0 ทันที
//   • แต่ "รายการ" ในลิสต์ยังอยู่ครบตามข้อมูลจริงใน DB — ไม่ได้ถูกซ่อน
//     รายการจะหายก็ต่อเมื่อใบถูก "ต่ออายุ/แก้ไขข้อมูล" จนไม่เข้าเกณฑ์เตือนแล้วเท่านั้น
//   • ถ้ามีใบใหม่เข้าเกณฑ์เตือนเพิ่ม (key ใหม่ที่ยังไม่เคยเห็น) badge จะเด้งขึ้นใหม่
//
// จำ "ที่เคยเห็นแล้ว" ผ่าน localStorage (ผูกกับ key = สถานะ+วันหมดอายุ ของแต่ละใบ)
// -> พอสถานะเปลี่ยน (ใกล้หมด -> หมดอายุ) หรือวันหมดอายุเปลี่ยน (ต่ออายุ) key เปลี่ยน
//    ระบบจะถือเป็นของใหม่และเด้งเตือนอีกครั้งเอง
//
// แสดงเฉพาะ role LOG (คนเดียวที่เข้าถึงบัญชีใบอนุญาต) — role อื่นไม่เห็นและไม่ยิง API
// ─────────────────────────────────────────────────────────────────────────────

const POLL_MS = 60_000
const SEEN_KEY = 'iconfirm_wh_alert_seen' // { [prefixedKey]: seenAtISO } รายการที่ผู้ใช้เปิดดูแล้ว

// ── คลังจำ "รายการที่เปิดดูแล้ว" (seen) บน localStorage ──────────────────────
function readSeen() {
  try {
    const raw = localStorage.getItem(SEEN_KEY)
    const obj = raw ? JSON.parse(raw) : {}
    return obj && typeof obj === 'object' ? obj : {}
  } catch {
    return {}
  }
}
function writeSeen(map) {
  try {
    localStorage.setItem(SEEN_KEY, JSON.stringify(map))
  } catch {
    /* localStorage เต็ม/ปิด — ปล่อยผ่าน ไม่ให้กระดิ่งพัง */
  }
}
// key ที่ prefix ฝั่งไว้ กันชนกันระหว่างนำเข้า/ส่งออก
const impKey = (it) => 'imp:' + dismissKey(it)
const expKey = (it) => 'exp:' + exportDismissKey(it)

// ตัวเลือก filter จำนวนรายการล่าสุดที่แสดง
const LIMIT_OPTIONS = [
  { value: 2, label: '2' },
  { value: 5, label: '5' },
  { value: 10, label: '10' },
  { value: 'all', label: 'ทั้งหมด' },
]

export default function WHAlertBell() {
  const navigate = useAppNavigate()
  const [open, setOpen] = useState(false)

  const [impItems, setImpItems] = useState([])
  const [expItems, setExpItems] = useState([])
  const [impNoDate, setImpNoDate] = useState(0)
  const [expNoDate, setExpNoDate] = useState(0)
  const [loaded, setLoaded] = useState(false)
  const [hasNew, setHasNew] = useState(false)
  const [seen, setSeen] = useState(() => readSeen())

  // filter "แสดงล่าสุดกี่รายการ" ของแต่ละฝั่ง (ค่าเริ่มต้น = 5)
  const [impLimit, setImpLimit] = useState(5)
  const [expLimit, setExpLimit] = useState(5)

  const rootRef = useRef(null)

  const load = useCallback(async () => {
    // ดึงทั้งสองฝั่งพร้อมกัน — ฝั่งไหนพังก็ไม่ล้มอีกฝั่ง (กระดิ่งพังไม่ควรทำทั้งหน้าล้ม)
    const [imp, exp] = await Promise.allSettled([
      getImportLicenseAlerts({ onlyAlert: true }),
      getExportLicenseAlerts({ onlyAlert: true }),
    ])
    let impList = []
    let expList = []
    if (imp.status === 'fulfilled') {
      impList = imp.value?.items || []
      setImpItems(impList)
      setImpNoDate(imp.value?.counts?.noDate || 0)
    }
    if (exp.status === 'fulfilled') {
      expList = exp.value?.items || []
      setExpItems(expList)
      setExpNoDate(exp.value?.counts?.noDate || 0)
    }

    // เก็บกวาด seen: เก็บเฉพาะ key ที่ยังมีอยู่จริง (ใบที่ต่ออายุ/ลบ key เก่าหลุดไปเอง)
    setSeen((prev) => {
      const live = new Set([...impList.map(impKey), ...expList.map(expKey)])
      let changed = false
      const next = {}
      for (const [k, v] of Object.entries(prev)) {
        if (live.has(k)) next[k] = v
        else changed = true
      }
      if (changed) writeSeen(next)
      return changed ? next : prev
    })

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

  // แยกกลุ่ม หมดอายุ/ใกล้หมดอายุ ของแต่ละฝั่ง (แสดง "ทุกรายการจริง" ไม่มีการซ่อน)
  const isAlertStatus = (i) => i.Status === 'EXPIRED' || i.Status === 'EXPIRING'
  const imp = useMemo(() => {
    const list = impItems.filter(isAlertStatus)
    return {
      all: list,
      expired: list.filter((it) => it.Status === 'EXPIRED'),
      expiring: list.filter((it) => it.Status === 'EXPIRING'),
    }
  }, [impItems])
  const exp = useMemo(() => {
    const list = expItems.filter(isAlertStatus)
    return {
      all: list,
      expired: list.filter((it) => it.Status === 'EXPIRED'),
      expiring: list.filter((it) => it.Status === 'EXPIRING'),
    }
  }, [expItems])

  // ── ตัวเลข badge = จำนวนแจ้งเตือน "ที่ยังไม่ได้เปิดดู" (unseen) รวมสองฝั่ง ──
  const unseenCount = useMemo(() => {
    let n = 0
    for (const it of imp.all) if (!Object.prototype.hasOwnProperty.call(seen, impKey(it))) n++
    for (const it of exp.all) if (!Object.prototype.hasOwnProperty.call(seen, expKey(it))) n++
    return n
  }, [imp.all, exp.all, seen])

  useEffect(() => {
    if (!loaded) return
    setHasNew(unseenCount > 0)
  }, [loaded, unseenCount])

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

  // เปิดกระดิ่ง = รับรู้แจ้งเตือนทั้งหมด -> mark seen ทุก key ปัจจุบัน -> badge เป็น 0
  function markAllSeen() {
    const next = { ...readSeen() }
    const nowISO = new Date().toISOString()
    for (const it of imp.all) next[impKey(it)] = nowISO
    for (const it of exp.all) next[expKey(it)] = nowISO
    writeSeen(next)
    setSeen(next)
    setHasNew(false)
  }

  function toggle() {
    const next = !open
    setOpen(next)
    if (next) markAllSeen()
  }

  const goImport = () => {
    setOpen(false)
    navigate('/warehouse')
  }
  const goExport = () => {
    setOpen(false)
    navigate('/warehouse/export-license')
  }

  const badge = unseenCount
  const totalOutstanding = imp.all.length + exp.all.length

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

          {loaded && totalOutstanding === 0 && (
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
            limit={impLimit}
            onLimitChange={setImpLimit}
            noDate={impNoDate}
            noDateLabel="ยังไม่ได้ระบุวันที่ออกใบอนุญาต"
            expiringLabel="ใกล้หมดอายุ (ภายใน 30 วัน)"
            onOpen={goImport}
            getKey={impKey}
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
            limit={expLimit}
            onLimitChange={setExpLimit}
            noDate={expNoDate}
            noDateLabel="ยังไม่ได้ระบุวันหมดอายุ/ใบขน"
            expiringLabel="ใกล้หมดอายุ (ภายใน 7 วัน)"
            onOpen={goExport}
            getKey={expKey}
            renderMeta={(it) => (
              <>
                Exception License {it.ExceptionLicense || '—'}
                {it.DeclarationDate ? ` · ใบขน ${formatThaiDate(it.DeclarationDate)}` : ''}
              </>
            )}
            titleField={(it) => it.SerialNumber || '—'}
          />

          {totalOutstanding > 0 && (
            <div className="lab-resolve-note">
              ตัวเลขบนกระดิ่งคือจำนวนที่<strong>ยังไม่ได้เปิดดู</strong> — พอเปิดดูแล้วจะหายไป
              ส่วน<strong>รายการด้านบน</strong>จะยังอยู่จนกว่าใบจะถูก<strong>ต่ออายุ/แก้ไขข้อมูล</strong>จนพ้นเกณฑ์เตือน
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
  limit,
  onLimitChange,
  noDate,
  noDateLabel,
  expiringLabel,
  onOpen,
  getKey,
  renderMeta,
  titleField,
}) {
  const total = expired.length + expiring.length
  const themeClass = theme === 'export' ? ' lab-section-export' : ' lab-section-import'

  // เรียงด่วนสุดก่อน (DaysLeft น้อย/ติดลบมากขึ้นก่อน) แล้วตัดตาม filter "ล่าสุด N รายการ"
  const sortByUrgent = (a, b) => (a.DaysLeft ?? 0) - (b.DaysLeft ?? 0)
  const limitNum = limit === 'all' ? Infinity : Number(limit)
  const expiredShown = [...expired].sort(sortByUrgent).slice(0, limitNum)
  const remain = Math.max(0, limitNum - expiredShown.length)
  const expiringShown = [...expiring].sort(sortByUrgent).slice(0, remain)
  const shownCount = expiredShown.length + expiringShown.length
  const hiddenByLimit = total - shownCount

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

      {/* filter จำนวนรายการล่าสุดที่แสดง (2 / 5 / 10 / ทั้งหมด) */}
      {total > 0 && (
        <div className="lab-limit-filter" role="group" aria-label={`แสดงล่าสุดกี่รายการ (${title})`}>
          <span className="lab-limit-label">ล่าสุด</span>
          {LIMIT_OPTIONS.map((o) => (
            <button
              key={String(o.value)}
              type="button"
              className={'lab-limit-btn' + (limit === o.value ? ' is-active' : '')}
              onClick={() => onLimitChange(o.value)}
            >
              {o.label}
            </button>
          ))}
        </div>
      )}

      <div className="lab-list">
        {loaded && total === 0 && (
          <div className="lab-empty lab-empty-ok">
            <span className="lab-empty-dot" />
            ไม่มีรายการต้องจัดการ
          </div>
        )}

        {expiredShown.length > 0 && (
          <>
            <div className="lab-group-label lab-group-expired">หมดอายุแล้ว</div>
            {expiredShown.map((it) => (
              <Row key={getKey(it)} item={it} onOpen={onOpen} renderMeta={renderMeta} titleField={titleField} />
            ))}
          </>
        )}

        {expiringShown.length > 0 && (
          <>
            <div className="lab-group-label lab-group-expiring">{expiringLabel}</div>
            {expiringShown.map((it) => (
              <Row key={getKey(it)} item={it} onOpen={onOpen} renderMeta={renderMeta} titleField={titleField} />
            ))}
          </>
        )}

        {hiddenByLimit > 0 && (
          <button type="button" className="lab-more-btn" onClick={() => onLimitChange('all')}>
            + ดูอีก {hiddenByLimit} รายการ
          </button>
        )}
      </div>

      {noDate > 0 && (
        <div className="lab-foot-note">
          <ClockIcon className="size-4" />
          มี {noDate} ใบที่{noDateLabel} — เติมวันที่เพื่อให้ระบบเตือนอายุได้
        </div>
      )}
    </div>
  )
}

function Row({ item, onOpen, renderMeta, titleField }) {
  const isExpired = item.Status === 'EXPIRED'
  return (
    <div className="lab-item">
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
    </div>
  )
}
