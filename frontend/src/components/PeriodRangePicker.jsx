import DatePickerField from './DatePickerField.jsx';
import { CalendarDaysIcon, XMarkIcon } from './icons.jsx';
import { PERIOD_MODE_OPTIONS, PERIOD_ANCHOR_HINT, periodRangeLabel } from '../lib/dateRange.js';
import './PeriodRangePicker.css';

// ---------------------------------------------------------------------------
// ตัวเลือกช่วงวันที่ (หน้า QA และ Export License ใช้ตัวเดียวกัน)
//
// เลย์เอาต์: [หัวข้อ + ปุ่มโหมด] [คำแนะนำ + ช่องวันที่] [ป้ายช่วงที่เลือก ........ ปุ่มล้างช่วง]
//   - หัวข้อทุกช่องอยู่บรรทัดเดียวกัน และทุกปุ่ม/ช่องสูงเท่ากัน (--prp-h) จึงเรียงตรงกันทั้งแถว
//   - จอไม่พอ (iPad แนวตั้ง / Surface แนวตั้ง) ป้ายช่วง + ปุ่มล้างช่วง ขึ้นแถวใหม่ด้วยกัน
//     ป้ายชิดซ้าย ปุ่มชิดขวา ไม่กระจัดกระจายคนละบรรทัด
//   - ปุ่ม "ล้างช่วง" อยู่ในคอมโพเนนต์นี้ (ส่ง onClear มา) ทั้งสองหน้าจึงหน้าตาเหมือนกัน
// ---------------------------------------------------------------------------
export default function PeriodRangePicker({
  mode = 'all',
  onModeChange,
  anchor = '',
  onAnchorChange,
  // min / max = ช่วงวันที่ของข้อมูล — ยังรับไว้เพื่อไม่ให้หน้าที่เรียกใช้พัง แต่ไม่ได้จำกัดปฏิทินแล้ว
  // (ปฏิทินเลือกได้ พ.ศ. 2543 – 2580 เสมอ ดู CALENDAR_MIN_YMD / CALENDAR_MAX_YMD ใน lib/typedDate.js)
  min,
  max,
  label = 'ช่วงเวลา',
  countLabel,
  onClear,
  clearLabel = 'ล้างช่วง'
}) {
  const showAnchor = mode && mode !== 'all';
  const rangeLabel = periodRangeLabel(mode, anchor);
  return <div className="prp">
      <div className="prp-field prp-field-modes">
        <span className="prp-label">{label}</span>
        <div className="vr-tabs prp-modes" role="tablist">
          {PERIOD_MODE_OPTIONS.map(o => <button key={o.key} type="button" role="tab" aria-selected={mode === o.key} className={'vr-tab prp-mode' + (mode === o.key ? ' vr-tab-active' : '')} onClick={() => onModeChange?.(o.key)}>
              {o.label}
            </button>)}
        </div>
      </div>

      {showAnchor && <div className="prp-field prp-anchor">
          <span className="prp-label">{PERIOD_ANCHOR_HINT[mode] || 'เลือกวันอ้างอิง'}</span>
          {/* ไม่ส่งช่วงวันที่ของข้อมูล (min/max) ให้ปฏิทิน — ให้เลือกได้ทุกปี พ.ศ. 2543 – 2580 */}
          <DatePickerField value={anchor} onChange={onAnchorChange} placeholder="— เลือกวัน —" />
        </div>}

      {showAnchor && <div className="prp-summary">
          <div className="prp-chip" title="ช่วงที่จะแสดง/ส่งออก">
            <CalendarDaysIcon className="size-4" />
            <span className="prp-chip-range">{rangeLabel}</span>
            {countLabel && <span className="prp-chip-count">{countLabel}</span>}
          </div>
          {onClear && <button type="button" className="prp-clear" onClick={onClear}>
              <XMarkIcon className="size-4" />
              {clearLabel}
            </button>}
        </div>}
    </div>;
}
