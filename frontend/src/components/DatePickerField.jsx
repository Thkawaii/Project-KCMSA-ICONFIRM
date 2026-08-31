import { useEffect, useMemo, useRef, useState } from 'react';
import './DatePickerField.css';
const THAI_MONTHS = ['มกราคม', 'กุมภาพันธ์', 'มีนาคม', 'เมษายน', 'พฤษภาคม', 'มิถุนายน', 'กรกฎาคม', 'สิงหาคม', 'กันยายน', 'ตุลาคม', 'พฤศจิกายน', 'ธันวาคม'];
const THAI_WEEKDAYS = ['อา', 'จ', 'อ', 'พ', 'พฤ', 'ศ', 'ส'];
const pad2 = n => String(n).padStart(2, '0');
const toYMD = d => `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
const monthKey = (y, m) => y * 12 + m;
function parseYMD(ymd) {
  if (!ymd) return null;
  const [y, m, d] = ymd.split('-').map(Number);
  return {
    y,
    m: m - 1,
    d
  };
}
export default function DatePickerField({
  value,
  onChange,
  min,
  max,
  placeholder = 'เลือกวันที่',
  renderTrigger
}) {
  const [open, setOpen] = useState(false);
  const boxRef = useRef(null);
  const selected = parseYMD(value);
  const [view, setView] = useState(() => {
    const base = selected || (() => {
      const t = new Date();
      let ymd = toYMD(t);
      if (min && ymd < min) ymd = min;
      if (max && ymd > max) ymd = max;
      const p = parseYMD(ymd);
      return {
        y: p.y,
        m: p.m
      };
    })();
    return {
      y: base.y,
      m: base.m
    };
  });
  useEffect(() => {
    if (open && selected) setView({
      y: selected.y,
      m: selected.m
    });
  }, [open]);
  useEffect(() => {
    if (!open) return;
    function onOutside(e) {
      if (boxRef.current && !boxRef.current.contains(e.target)) setOpen(false);
    }
    function onKey(e) {
      if (e.key === 'Escape') setOpen(false);
    }
    document.addEventListener('mousedown', onOutside);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onOutside);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);
  const weeks = useMemo(() => {
    const first = new Date(view.y, view.m, 1);
    const start = new Date(view.y, view.m, 1 - first.getDay());
    const cells = [];
    for (let i = 0; i < 42; i++) {
      const d = new Date(start.getFullYear(), start.getMonth(), start.getDate() + i);
      cells.push(d);
    }
    const rows = [];
    for (let i = 0; i < 6; i++) rows.push(cells.slice(i * 7, i * 7 + 7));
    return rows;
  }, [view]);
  const todayYMD = toYMD(new Date());
  const viewKey = monthKey(view.y, view.m);
  const minKey = min ? (() => {
    const p = parseYMD(min);
    return monthKey(p.y, p.m);
  })() : null;
  const maxKey = max ? (() => {
    const p = parseYMD(max);
    return monthKey(p.y, p.m);
  })() : null;
  const prevDisabled = minKey !== null && viewKey <= minKey;
  const nextDisabled = maxKey !== null && viewKey >= maxKey;
  function goMonth(delta) {
    setView(v => {
      const d = new Date(v.y, v.m + delta, 1);
      return {
        y: d.getFullYear(),
        m: d.getMonth()
      };
    });
  }
  function pick(d) {
    onChange(toYMD(d));
    setOpen(false);
  }
  const triggerLabel = selected ? `${selected.d} ${THAI_MONTHS[selected.m]} ${selected.y + 543}` : placeholder;
  return <div className="dpf" ref={boxRef}>
      {renderTrigger ? renderTrigger({
      open,
      toggle: () => setOpen(o => !o),
      label: triggerLabel,
      hasValue: !!selected
    }) : <button type="button" className={'dpf-trigger' + (open ? ' dpf-trigger-open' : '')} onClick={() => setOpen(o => !o)} aria-haspopup="dialog" aria-expanded={open}>
          <span className={selected ? 'dpf-value' : 'dpf-value dpf-value-empty'}>
            {triggerLabel}
          </span>
          <span className="dpf-cal-icon" aria-hidden="true">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
              <rect x="3" y="4.5" width="18" height="16" rx="2.5" stroke="currentColor" strokeWidth="1.6" />
              <path d="M3 9h18M8 3v3M16 3v3" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
            </svg>
          </span>
        </button>}

      {open && <div className="dpf-pop" role="dialog">
          <div className="dpf-head">
            <button type="button" className="dpf-nav" onClick={() => goMonth(-1)} disabled={prevDisabled} aria-label="เดือนก่อนหน้า">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                <path d="M15 6l-6 6 6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </button>
            <div className="dpf-title">
              {THAI_MONTHS[view.m]} {view.y + 543}
            </div>
            <button type="button" className="dpf-nav" onClick={() => goMonth(1)} disabled={nextDisabled} aria-label="เดือนถัดไป">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                <path d="M9 6l6 6-6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </button>
          </div>

          <div className="dpf-weekdays">
            {THAI_WEEKDAYS.map((w, i) => <div key={i} className="dpf-weekday">{w}</div>)}
          </div>

          <div className="dpf-grid">
            {weeks.map((row, ri) => row.map((d, ci) => {
          const ymd = toYMD(d);
          const inMonth = d.getMonth() === view.m;
          const isSelected = value === ymd;
          const isToday = todayYMD === ymd;
          const disabled = min && ymd < min || max && ymd > max;
          return <button key={`${ri}-${ci}`} type="button" className={'dpf-day' + (inMonth ? '' : ' dpf-day-out') + (isSelected ? ' dpf-day-selected' : '') + (isToday && !isSelected ? ' dpf-day-today' : '')} disabled={disabled} onClick={() => pick(d)}>
                    {d.getDate()}
                  </button>;
        }))}
          </div>
        </div>}
    </div>;
}
