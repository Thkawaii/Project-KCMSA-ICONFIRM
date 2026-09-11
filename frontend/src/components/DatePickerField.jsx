import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { CALENDAR_MAX_YMD, CALENDAR_MIN_YMD, THAI_MONTHS, THAI_MONTHS_SHORT, formatShortThaiDate, formatTypedDate, parseTypedDate, parseYMD, ymdOf } from '../lib/typedDate.js';
import './DatePickerField.css';

// ---------------------------------------------------------------------------
// ช่องเลือกวันที่ (ใช้ในตัวกรองช่วงวันที่ของหน้า QA และ Export License)
//
// ใช้ได้ 2 ทาง
//   1. พิมพ์วันที่เองในช่อง เช่น 15/07/2569 แล้วกด Enter (หรือคลิกออก)
//      รับทั้งปี พ.ศ./ค.ศ. ปี 2 หลัก ตัวเลขล้วน และชื่อเดือนไทย — ดู lib/typedDate.js
//   2. เปิดปฏิทิน — กดที่ชื่อเดือน/ปีบนหัวปฏิทินเพื่อกระโดดไปเดือนหรือปีที่ต้องการได้ทันที
//      (เดิมต้องกด < > ทีละเดือน กว่าจะย้อนไปปีก่อน ๆ ต้องกดหลายสิบครั้ง)
//
// iPad / มือถือ: แตะช่องพิมพ์จะขึ้นแป้นตัวเลข แต่ไม่เด้งปฏิทินทับ
// ถ้าอยากเลือกจากปฏิทินให้แตะไอคอนปฏิทินด้านขวา
//
// ตำแหน่งปฏิทิน: วัดพื้นที่ว่างจริงบนจอทุกครั้งที่เปิด (รวม iPad / Surface / จอโน้ตบุ๊กที่ซูม 125%)
//   ใต้ช่องพอ → เปิดด้านล่าง | ไม่พอแต่ด้านบนพอ → เปิดขึ้นด้านบน
//   ไม่พอทั้งคู่ → เลื่อนหน้าให้ปฏิทินอยู่ในจอ ถ้ายังไม่พออีกให้เลื่อนภายในปฏิทินแทนการล้นจอ
//   ล้นขอบขวา → ขยับเข้ามาในจอ
// ---------------------------------------------------------------------------

const THAI_WEEKDAYS = ['อา', 'จ', 'อ', 'พ', 'พฤ', 'ศ', 'ส'];
const pad2 = n => String(n).padStart(2, '0');
const toYMD = d => `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
const monthKey = (y, m) => y * 12 + m;
function CalendarIcon() {
  return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <rect x="3" y="4.5" width="18" height="16" rx="2.5" stroke="currentColor" strokeWidth="1.6" />
      <path d="M3 9h18M8 3v3M16 3v3" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>;
}
function Chevron({
  dir
}) {
  const d = dir === 'left' ? 'M15 6l-6 6 6 6' : dir === 'right' ? 'M9 6l6 6-6 6' : 'M6 9l6 6 6-6';
  return <svg width={dir === 'down' ? 14 : 18} height={dir === 'down' ? 14 : 18} viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d={d} stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>;
}
// เลื่อนกล่องที่เลื่อนได้ตัวแรกที่ครอบอยู่ (หรือทั้งหน้า) ลงไป delta px
function scrollAncestorsBy(el, delta) {
  let node = el.parentElement;
  while (node && node !== document.body) {
    const style = getComputedStyle(node);
    const scrollable = /(auto|scroll)/.test(style.overflowY) && node.scrollHeight > node.clientHeight;
    if (scrollable) {
      const before = node.scrollTop;
      node.scrollTop = before + delta;
      if (node.scrollTop !== before) return;
    }
    node = node.parentElement;
  }
  const root = document.scrollingElement || document.documentElement;
  root.scrollTop += delta;
}
export default function DatePickerField({
  value,
  onChange,
  min: minProp,
  max: maxProp,
  placeholder = 'เลือกวันที่',
  renderTrigger
}) {
  // ปฏิทินเลือกได้ตั้งแต่ พ.ศ. 2543 ถึง 2580 เสมอ — ถ้าผู้เรียกส่ง min/max มา จะแคบลงได้แต่ไม่กว้างกว่านี้
  const min = minProp && minProp > CALENDAR_MIN_YMD ? minProp : CALENDAR_MIN_YMD;
  const max = maxProp && maxProp < CALENDAR_MAX_YMD ? maxProp : CALENDAR_MAX_YMD;
  const [open, setOpen] = useState(false);
  // days = ตารางวัน, months = เลือกเดือน, years = เลือกปี
  const [panel, setPanel] = useState('days');
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState('');
  const [error, setError] = useState('');
  const boxRef = useRef(null);
  const inputRef = useRef(null);
  const yearListRef = useRef(null);
  const popRef = useRef(null);
  // up = เปิดขึ้นด้านบน, shiftX = ขยับซ้าย/ขวาไม่ให้ล้นจอ, maxH = จำกัดความสูงเมื่อจอเตี้ยมาก
  const [placement, setPlacement] = useState({
    up: false,
    shiftX: 0,
    maxH: null
  });
  const [errorUp, setErrorUp] = useState(false);
  const textRef = useRef('');
  const pointerTypeRef = useRef('mouse');
  const errorTimerRef = useRef(null);
  const selected = parseYMD(value);
  // ค่าล่าสุดที่ส่งออกไปแล้ว — กันเรียก onChange ซ้ำตอนเลือกวันแล้วช่องพิมพ์หลุดโฟกัสตามมา
  const lastValueRef = useRef(value);
  lastValueRef.current = value;
  const todayYMD = toYMD(new Date());
  function clampYMD(ymd) {
    if (min && ymd < min) return min;
    if (max && ymd > max) return max;
    return ymd;
  }
  const [view, setView] = useState(() => {
    const base = selected || parseYMD(clampYMD(todayYMD));
    return {
      y: base.y,
      m: base.m
    };
  });
  useEffect(() => {
    if (!open) return;
    setPanel('days');
    if (selected) setView({
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
  useEffect(() => () => clearTimeout(errorTimerRef.current), []);
  useLayoutEffect(() => {
    if (!open) return undefined;
    const MARGIN = 8;
    const GAP = 6;
    function viewportBox() {
      const vv = window.visualViewport;
      return vv ? {
        top: vv.offsetTop,
        left: vv.offsetLeft,
        width: vv.width,
        height: vv.height
      } : {
        top: 0,
        left: 0,
        width: window.innerWidth,
        height: window.innerHeight
      };
    }
    function compute() {
      const pop = popRef.current;
      const box = boxRef.current;
      if (!pop || !box) return null;
      const vp = viewportBox();
      const r = box.getBoundingClientRect();
      const h = pop.scrollHeight + 2; // + เส้นขอบ
      const w = pop.offsetWidth;
      const below = vp.top + vp.height - r.bottom - GAP - MARGIN;
      const above = r.top - vp.top - GAP - MARGIN;
      let shiftX = 0;
      const limitRight = vp.left + vp.width - MARGIN;
      if (r.left + w > limitRight) shiftX = limitRight - (r.left + w);
      if (r.left + shiftX < vp.left + MARGIN) shiftX = vp.left + MARGIN - r.left;
      return {
        h,
        below,
        above,
        shiftX
      };
    }
    function place(allowScroll) {
      let m = compute();
      if (!m) return;
      if (m.h <= m.below) return setPlacement({
        up: false,
        shiftX: m.shiftX,
        maxH: null
      });
      if (m.h <= m.above) return setPlacement({
        up: true,
        shiftX: m.shiftX,
        maxH: null
      });
      if (allowScroll) {
        // ไม่พอทั้งบนและล่าง — เลื่อนหน้าให้ช่องวันที่ขึ้นไปใกล้ขอบบน แล้ววัดใหม่
        const need = m.h - m.below;
        const canMoveUp = Math.max(0, m.above - 4);
        const delta = Math.min(need, canMoveUp);
        if (delta > 0) {
          const before = boxRef.current.getBoundingClientRect().top;
          scrollAncestorsBy(boxRef.current, delta);
          const moved = before - boxRef.current.getBoundingClientRect().top;
          if (moved > 0) m = compute() || m;
          if (m.h <= m.below) return setPlacement({
            up: false,
            shiftX: m.shiftX,
            maxH: null
          });
        }
      }
      const up = m.above > m.below;
      setPlacement({
        up,
        shiftX: m.shiftX,
        maxH: Math.max(200, Math.floor(up ? m.above : m.below))
      });
    }
    place(true);
    const onResize = () => place(false);
    window.addEventListener('resize', onResize);
    window.visualViewport?.addEventListener('resize', onResize);
    return () => {
      window.removeEventListener('resize', onResize);
      window.visualViewport?.removeEventListener('resize', onResize);
    };
  }, [open, panel]);

  // เปิดหน้าเลือกปีแล้ว เลื่อนให้ปีที่กำลังดูอยู่กลางกล่อง ไม่ต้องไล่หาเอง
  useEffect(() => {
    if (panel !== 'years' || !yearListRef.current) return;
    const list = yearListRef.current;
    const active = list.querySelector('.dpf-cell-active');
    // เลื่อนเฉพาะในกล่องรายการปี (ไม่ใช้ scrollIntoView เพราะจะพาทั้งหน้าเลื่อนตามไปด้วย)
    if (active) list.scrollTop = active.offsetTop - list.clientHeight / 2 + active.offsetHeight / 2;
  }, [panel]);
  const weeks = useMemo(() => {
    const first = new Date(view.y, view.m, 1);
    const start = new Date(view.y, view.m, 1 - first.getDay());
    const rows = [];
    for (let w = 0; w < 6; w++) {
      const row = [];
      for (let i = 0; i < 7; i++) row.push(new Date(start.getFullYear(), start.getMonth(), start.getDate() + w * 7 + i));
      rows.push(row);
    }
    return rows;
  }, [view]);
  const minP = parseYMD(min);
  const maxP = parseYMD(max);
  const minKey = minP ? monthKey(minP.y, minP.m) : null;
  const maxKey = maxP ? monthKey(maxP.y, maxP.m) : null;
  const viewKey = monthKey(view.y, view.m);
  const prevDisabled = minKey !== null && viewKey <= minKey;
  const nextDisabled = maxKey !== null && viewKey >= maxKey;
  // หน้าเลือกปีแสดงทุกปีในช่วง min–max (ค่าเริ่มต้น พ.ศ. 2543 – 2580)
  const firstYear = minP.y;
  const lastYear = maxP.y;
  const years = useMemo(() => {
    const list = [];
    for (let y = firstYear; y <= lastYear; y++) list.push(y);
    return list;
  }, [firstYear, lastYear]);
  const monthDisabled = (y, m) => minKey !== null && monthKey(y, m) < minKey || maxKey !== null && monthKey(y, m) > maxKey;
  function goMonth(delta) {
    setView(v => {
      const d = new Date(v.y, v.m + delta, 1);
      return {
        y: d.getFullYear(),
        m: d.getMonth()
      };
    });
  }
  function goYear(delta) {
    setView(v => ({
      y: Math.min(Math.max(v.y + delta, firstYear), lastYear),
      m: v.m
    }));
  }
  function chooseYear(y) {
    // เดือนเดิมอาจอยู่นอกช่วงข้อมูลของปีใหม่ — ขยับเข้าช่วงให้ แล้วไปเลือกเดือนต่อ
    let m = view.m;
    if (minKey !== null && monthKey(y, m) < minKey) m = minP.m;
    if (maxKey !== null && monthKey(y, m) > maxKey) m = maxP.m;
    setView({
      y,
      m
    });
    setPanel('months');
  }
  function chooseMonth(m) {
    setView(v => ({
      y: v.y,
      m
    }));
    setPanel('days');
  }
  function flashError(message) {
    // ช่องอยู่ติดขอบล่างจอ — ให้ข้อความแจ้งเตือนขึ้นด้านบนแทน จะได้ไม่ถูกตัด
    const r = boxRef.current?.getBoundingClientRect();
    const vh = window.visualViewport?.height || window.innerHeight;
    setErrorUp(!!r && vh - r.bottom < 90 && r.top > 90);
    setError(message);
    clearTimeout(errorTimerRef.current);
    errorTimerRef.current = setTimeout(() => setError(''), 4000);
  }
  function rangeMessage() {
    return `เลือกได้ระหว่าง ${formatShortThaiDate(min)} – ${formatShortThaiDate(max)}`;
  }

  // ตรวจข้อความที่พิมพ์ — คืน true เมื่อบันทึกค่าได้
  function commitText(raw) {
    const trimmed = String(raw ?? '').trim();
    if (!trimmed) return true; // ลบข้อความทิ้ง = ไม่เปลี่ยนค่าเดิม
    const parsed = parseTypedDate(trimmed, {
      min,
      max
    });
    if (!parsed.ymd) {
      flashError('วันที่ไม่ถูกต้อง — พิมพ์แบบ วว/ดด/ปปปป เช่น 15/07/2569');
      return false;
    }
    if (min && parsed.ymd < min || max && parsed.ymd > max) {
      flashError(rangeMessage());
      return false;
    }
    setError('');
    if (parsed.ymd !== lastValueRef.current) {
      lastValueRef.current = parsed.ymd;
      onChange(parsed.ymd);
    }
    return true;
  }
  function setTextBoth(next) {
    textRef.current = next;
    setText(next);
  }
  function handleFocus() {
    setEditing(true);
    setTextBoth(formatTypedDate(value));
    requestAnimationFrame(() => inputRef.current?.select());
    // เมาส์: เปิดปฏิทินให้เลย / นิ้ว (iPad): ไม่เปิด เพราะแป้นพิมพ์กับปฏิทินจะทับกัน
    if (pointerTypeRef.current !== 'touch') setOpen(true);
  }
  function handleTextChange(e) {
    // ไม่เติม "/" ให้อัตโนมัติระหว่างพิมพ์ — จะไปขัดคนที่พิมพ์ "/" เอง หรือพิมพ์แบบ 2026-07-15 / 15 ก.ค. 2569
    // พิมพ์ตัวเลขล้วน 15072569 ก็ได้ พอกด Enter หรือคลิกออก ระบบจัดรูปให้เอง
    const next = e.target.value;
    setTextBoth(next);
    if (error) setError('');
    // พิมพ์ครบเป็นวันที่แล้ว เลื่อนปฏิทินไปเดือนนั้นให้เห็นทันที
    const parsed = parseTypedDate(next, {
      min,
      max
    });
    const p = parseYMD(parsed.ymd);
    if (p) {
      setView({
        y: p.y,
        m: p.m
      });
      setPanel('days');
    }
  }
  function handleBlur() {
    // ข้อความผิดตอนคลิกออก: คืนค่าเดิม และแจ้งเตือนสั้น ๆ ใต้ช่อง
    commitText(textRef.current);
    setEditing(false);
  }
  function handleKeyDown(e) {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (commitText(textRef.current)) {
        setOpen(false);
        inputRef.current?.blur();
      }
    } else if (e.key === 'ArrowDown' && !open) {
      e.preventDefault();
      setOpen(true);
    }
  }
  function pick(ymd) {
    setError('');
    setTextBoth(formatTypedDate(ymd));
    if (ymd !== lastValueRef.current) {
      lastValueRef.current = ymd;
      onChange(ymd);
    }
    setOpen(false);
    if (document.activeElement === inputRef.current) inputRef.current.blur();
  }
  const triggerLabel = selected ? `${selected.d} ${THAI_MONTHS[selected.m]} ${selected.y + 543}` : placeholder;
  const todayDisabled = min && todayYMD < min || max && todayYMD > max;
  // data-scan-ignore: หน้าที่ฟังเครื่องยิงบาร์โค้ดจากคีย์บอร์ด (WH / MFG) จะไม่ดักการพิมพ์ในช่องนี้
  return <div className="dpf" ref={boxRef} data-scan-ignore="">
      {renderTrigger ? renderTrigger({
      open,
      toggle: () => setOpen(o => !o),
      label: triggerLabel,
      hasValue: !!selected
    }) : <div className={'dpf-trigger' + (open || editing ? ' dpf-trigger-open' : '') + (error ? ' dpf-trigger-error' : '')} onPointerDown={e => {
      pointerTypeRef.current = e.pointerType || 'mouse';
    }}>
          <input ref={inputRef} className={'dpf-input' + (selected || editing ? '' : ' dpf-input-empty')} type="text" inputMode="numeric" autoComplete="off" spellCheck={false} value={editing ? text : selected ? triggerLabel : ''} placeholder={editing ? 'วว/ดด/ปปปป' : placeholder} onFocus={handleFocus} onChange={handleTextChange} onBlur={handleBlur} onKeyDown={handleKeyDown} aria-label="วันที่ (พิมพ์ได้ เช่น 15/07/2569)" aria-invalid={!!error} />
          <button type="button" className="dpf-cal-btn" onMouseDown={e => e.preventDefault()} onClick={() => setOpen(o => !o)} aria-haspopup="dialog" aria-expanded={open} aria-label="เปิดปฏิทิน" title="เลือกจากปฏิทิน">
            <CalendarIcon />
          </button>
        </div>}

      {error && !open && <div className={'dpf-error' + (errorUp ? ' dpf-error-up' : '')} role="alert">{error}</div>}

      {open && <div ref={popRef} className={'dpf-pop' + (placement.up ? ' dpf-pop-up' : '') + (placement.maxH ? ' dpf-pop-scroll' : '')} style={{
      left: placement.shiftX || 0,
      maxHeight: placement.maxH || undefined
    }} role="dialog"
    // กดปุ่มในปฏิทินแล้วไม่ให้ช่องพิมพ์หลุดโฟกัส (ไม่งั้นข้อความที่พิมพ์ค้างจะถูกตรวจก่อนเลือกวัน)
    onMouseDown={e => e.preventDefault()}>
          {error && <div className="dpf-error dpf-error-inline" role="alert">{error}</div>}

          {panel === 'days' && <>
              <div className="dpf-head">
                <button type="button" className="dpf-nav" onClick={() => goMonth(-1)} disabled={prevDisabled} aria-label="เดือนก่อนหน้า">
                  <Chevron dir="left" />
                </button>
                <div className="dpf-title-group">
                  <button type="button" className="dpf-title-btn" onClick={() => setPanel('months')} aria-label="เลือกเดือน">
                    {THAI_MONTHS[view.m]}
                    <Chevron dir="down" />
                  </button>
                  <button type="button" className="dpf-title-btn" onClick={() => setPanel('years')} aria-label="เลือกปี">
                    {view.y + 543}
                    <Chevron dir="down" />
                  </button>
                </div>
                <button type="button" className="dpf-nav" onClick={() => goMonth(1)} disabled={nextDisabled} aria-label="เดือนถัดไป">
                  <Chevron dir="right" />
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
              return <button key={`${ri}-${ci}`} type="button" className={'dpf-day' + (inMonth ? '' : ' dpf-day-out') + (isSelected ? ' dpf-day-selected' : '') + (isToday && !isSelected ? ' dpf-day-today' : '')} disabled={disabled} onClick={() => pick(ymd)}>
                        {d.getDate()}
                      </button>;
            }))}
              </div>
            </>}

          {panel === 'months' && <>
              <div className="dpf-head">
                <button type="button" className="dpf-nav" onClick={() => goYear(-1)} disabled={view.y <= firstYear} aria-label="ปีก่อนหน้า">
                  <Chevron dir="left" />
                </button>
                <button type="button" className="dpf-title-btn" onClick={() => setPanel('years')} aria-label="เลือกปี">
                  ปี {view.y + 543}
                  <Chevron dir="down" />
                </button>
                <button type="button" className="dpf-nav" onClick={() => goYear(1)} disabled={view.y >= lastYear} aria-label="ปีถัดไป">
                  <Chevron dir="right" />
                </button>
              </div>
              <div className="dpf-cells dpf-cells-months">
                {THAI_MONTHS_SHORT.map((label, m) => {
              const isActive = m === view.m;
              const isSelected = selected && selected.y === view.y && selected.m === m;
              return <button key={label} type="button" className={'dpf-cell' + (isActive ? ' dpf-cell-active' : '') + (isSelected ? ' dpf-cell-selected' : '')} disabled={monthDisabled(view.y, m)} onClick={() => chooseMonth(m)} title={THAI_MONTHS[m]}>
                      {label}
                    </button>;
            })}
              </div>
            </>}

          {panel === 'years' && <>
              <div className="dpf-head dpf-head-center">
                <button type="button" className="dpf-title-btn" onClick={() => setPanel('days')} aria-label="กลับไปหน้าปฏิทิน">
                  เลือกปี พ.ศ. {firstYear + 543} – {lastYear + 543}
                </button>
              </div>
              <div className="dpf-cells dpf-cells-years" ref={yearListRef}>
                {years.map(y => {
              const isActive = y === view.y;
              const isSelected = selected && selected.y === y;
              return <button key={y} type="button" className={'dpf-cell' + (isActive ? ' dpf-cell-active' : '') + (isSelected ? ' dpf-cell-selected' : '')} onClick={() => chooseYear(y)} title={`ค.ศ. ${y}`}>
                      {y + 543}
                    </button>;
            })}
              </div>
            </>}

          <div className="dpf-foot">
            <span className="dpf-hint">พิมพ์ได้ เช่น 15/07/2569</span>
            <button type="button" className="dpf-today" onClick={() => pick(todayYMD)} disabled={!!todayDisabled}>
              วันนี้
            </button>
          </div>
        </div>}
    </div>;
}
