import { useEffect, useRef, useState } from 'react'

// dropdown ที่วาดเองทั้งหมด ใช้แทน <select>
//
// เหตุผล: <select> ของ iOS/Android วาด popup ด้วยขนาดของ OS เอง
// CSS คุมไม่ได้เลย บนมือถือเลยออกมาตัวใหญ่และล้นกรอบการ์ด
//
// props:
//   value       — ค่าที่เลือกอยู่
//   onChange    — (value) => void
//   options     — [{ value, label }]
//   placeholder — ข้อความตอนยังไม่เลือก
export default function SelectField({
  value,
  onChange,
  options = [],
  placeholder = '— เลือก —',
  disabled = false,
  id,
}) {
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(0)
  const boxRef = useRef(null)
  const listRef = useRef(null)

  const selected = options.find((o) => o.value === value)

  useEffect(() => {
    if (!open) return

    function onOutside(e) {
      if (boxRef.current && !boxRef.current.contains(e.target)) setOpen(false)
    }

    document.addEventListener('mousedown', onOutside)
    return () => document.removeEventListener('mousedown', onOutside)
  }, [open])

  useEffect(() => {
    if (open) setActive(Math.max(options.findIndex((o) => o.value === value), 0))
  }, [open, value, options])

  function choose(option) {
    onChange(option.value)
    setOpen(false)
  }

  function onKeyDown(e) {
    if (disabled) return

    if (e.key === 'Escape') return setOpen(false)

    if (!open && (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown')) {
      e.preventDefault()
      return setOpen(true)
    }

    if (!open) return

    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActive((i) => Math.min(i + 1, options.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActive((i) => Math.max(i - 1, 0))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (options[active]) choose(options[active])
    }
  }

  return (
    <div className="sf" ref={boxRef}>
      <button
        type="button"
        id={id}
        className={'sf-trigger' + (open ? ' sf-trigger-open' : '')}
        onClick={() => !disabled && setOpen(!open)}
        onKeyDown={onKeyDown}
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
      >
        <span className={selected ? 'sf-value' : 'sf-value sf-value-empty'}>
          {selected ? selected.label : placeholder}
        </span>
        <span className="sf-chevron" aria-hidden="true">
          <svg width="12" height="8" viewBox="0 0 12 8" fill="none">
            <path
              d="M1 1.5L6 6.5L11 1.5"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </span>
      </button>

      {open && (
        <ul className="sf-list" role="listbox" ref={listRef}>
          {options.map((option, i) => (
            <li key={option.value}>
              <button
                type="button"
                role="option"
                aria-selected={option.value === value}
                className={
                  'sf-option' +
                  (option.value === value ? ' sf-option-selected' : '') +
                  (i === active ? ' sf-option-active' : '')
                }
                onMouseEnter={() => setActive(i)}
                onClick={() => choose(option)}
              >
                {option.label}
              </button>
            </li>
          ))}

          {!options.length && <li className="sf-empty">ไม่มีตัวเลือก</li>}
        </ul>
      )}
    </div>
  )
}