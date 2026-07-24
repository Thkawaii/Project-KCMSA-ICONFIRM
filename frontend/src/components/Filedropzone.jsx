import { useRef, useState } from 'react'

// ปุ่มเลือกไฟล์แบบลากมาวางได้ ใช้ร่วมกันทั้งหน้า Warehouse และ IT Controller
//
// props:
//   file      — ไฟล์ที่เลือกอยู่ (File | null) — คุมจากข้างนอก
//   onSelect  — เรียกเมื่อผู้ใช้เลือก/ลากไฟล์มาวาง (file) => void
//   accept    — เช่น '.xlsx,.xls' หรือ '.pdf'
//   label     — หัวข้อ เช่น 'อัปโหลด SO / สต็อกเข้าคลัง'
//   hint      — คำอธิบายชนิดไฟล์
//   disabled  — ปิดระหว่างอัปโหลด
export default function FileDropZone({
  file,
  onSelect,
  accept = '',
  label = 'เลือกไฟล์',
  hint = '',
  disabled = false,
}) {
  const inputRef = useRef(null)
  const [dragging, setDragging] = useState(false)

  const extensions = accept
    .split(',')
    .map((a) => a.trim().replace('.', '').toUpperCase())
    .filter(Boolean)

  function accepts(candidate) {
    if (!extensions.length) return true
    const ext = candidate.name.split('.').pop()?.toUpperCase()
    return extensions.includes(ext)
  }

  function pick(candidate) {
    if (!candidate || disabled) return
    if (!accepts(candidate)) return
    onSelect(candidate)
  }

  function handleDrop(e) {
    e.preventDefault()
    setDragging(false)
    pick(e.dataTransfer.files?.[0])
  }

  function openPicker() {
    if (!disabled) inputRef.current?.click()
  }

  const className = [
    'fdz',
    dragging ? 'fdz-dragging' : '',
    file ? 'fdz-filled' : '',
    disabled ? 'fdz-disabled' : '',
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <div
      className={className}
      onClick={openPicker}
      onDragOver={(e) => {
        e.preventDefault()
        if (!disabled) setDragging(true)
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={handleDrop}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          openPicker()
        }
      }}
      role="button"
      tabIndex={disabled ? -1 : 0}
      aria-label={label}
    >
      <span className={file ? 'fdz-badge ' + badgeTone(file.name) : 'fdz-icon'}>
        {file ? extOf(file.name) : <UploadIcon />}
      </span>

      <span className="fdz-body">
        {file ? (
          <>
            <span className="fdz-name">{file.name}</span>
            <span className="fdz-meta">{formatSize(file.size)} · กดเพื่อเปลี่ยนไฟล์</span>
          </>
        ) : (
          <>
            <span className="fdz-label">{label}</span>
            <span className="fdz-meta">
              {hint || (extensions.length ? `ลากไฟล์มาวาง หรือกดเพื่อเลือก · ${extensions.join(' / ')}` : 'ลากไฟล์มาวาง หรือกดเพื่อเลือก')}
            </span>
          </>
        )}
      </span>

      {!file && !disabled && <span className="fdz-cta">เลือกไฟล์</span>}

      {file && !disabled && (
        <button
          type="button"
          className="fdz-clear"
          aria-label="เอาไฟล์ออก"
          onClick={(e) => {
            e.stopPropagation()
            onSelect(null)
            if (inputRef.current) inputRef.current.value = ''
          }}
        >
          ✕
        </button>
      )}

      <input
        ref={inputRef}
        type="file"
        accept={accept}
        className="fdz-input"
        disabled={disabled}
        onChange={(e) => {
          pick(e.target.files?.[0])
          e.target.value = ''
        }}
      />
    </div>
  )
}

function formatSize(bytes) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function extOf(name) {
  return (name.split('.').pop() || '').toUpperCase().slice(0, 4)
}

function badgeTone(name) {
  return extOf(name) === 'PDF' ? 'fdz-badge-pdf' : 'fdz-badge-xls'
}

function UploadIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M7 18a4.5 4.5 0 01-.4-8.98A5.5 5.5 0 0117 8.5a4 4 0 01-.5 7.98"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M12 19v-7m0 0l-2.5 2.5M12 12l2.5 2.5"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}