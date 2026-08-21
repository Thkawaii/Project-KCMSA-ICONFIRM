import DatePickerField from './DatePickerField.jsx'
import { CalendarDaysIcon } from './icons.jsx'
import {
  PERIOD_MODE_OPTIONS,
  PERIOD_ANCHOR_HINT,
  periodRangeLabel,
} from '../lib/dateRange.js'
import './PeriodRangePicker.css'

// ตัวเลือกช่วงเวลาใช้ซ้ำได้ — โหมด (ทั้งหมด/รายวัน/รายสัปดาห์/รายเดือน/รายปี)
// + ปฏิทินเลือก "วันอ้างอิง" (anchor) เพื่อระบุว่าจะดู/Export ของช่วงไหน
//
// props:
//   mode           'all' | 'day' | 'week' | 'month' | 'year'
//   onModeChange   (mode) => void
//   anchor         'YYYY-MM-DD' — วันอ้างอิง (โหมด != all ต้องมี)
//   onAnchorChange (ymd) => void
//   min / max      'YYYY-MM-DD' ขอบเขตปฏิทิน (ไม่ใส่ = ไม่จำกัด)
//   label          ป้ายหัวข้อด้านซ้าย (เช่น "ช่วงเวลา")
//   countLabel     ข้อความจำนวนผลลัพธ์ (แสดงต่อท้ายชิปช่วง) — ไม่ใส่ก็ได้
export default function PeriodRangePicker({
  mode = 'all',
  onModeChange,
  anchor = '',
  onAnchorChange,
  min,
  max,
  label = 'ช่วงเวลา',
  countLabel,
}) {
  const showAnchor = mode && mode !== 'all'
  const rangeLabel = periodRangeLabel(mode, anchor)

  return (
    <div className="prp">
      <div className="prp-field">
        <span className="prp-label">{label}</span>
        <div className="vr-tabs prp-modes" role="tablist">
          {PERIOD_MODE_OPTIONS.map((o) => (
            <button
              key={o.key}
              type="button"
              role="tab"
              aria-selected={mode === o.key}
              className={'vr-tab prp-mode' + (mode === o.key ? ' vr-tab-active' : '')}
              onClick={() => onModeChange?.(o.key)}
            >
              {o.label}
            </button>
          ))}
        </div>
      </div>

      {showAnchor && (
        <div className="prp-field prp-anchor">
          <span className="prp-label">{PERIOD_ANCHOR_HINT[mode] || 'เลือกวันอ้างอิง'}</span>
          <DatePickerField
            value={anchor}
            onChange={onAnchorChange}
            min={min}
            max={max}
            placeholder="— เลือกวัน —"
          />
        </div>
      )}

      {showAnchor && (
        <div className="prp-chip" title="ช่วงที่จะแสดง/ส่งออก">
          <CalendarDaysIcon className="size-4" />
          <span className="prp-chip-range">{rangeLabel}</span>
          {countLabel && <span className="prp-chip-count">{countLabel}</span>}
        </div>
      )}
    </div>
  )
}
