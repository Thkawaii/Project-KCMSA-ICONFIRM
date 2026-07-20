import { useEffect, useRef, useState } from 'react'

const SCANNER_ELEMENT_ID = 'barcode-scanner-viewport'

export default function BarcodeScannerModal({ title, onDetected, onClose }) {
  const scannerRef = useRef(null)
  const [error, setError] = useState('')
  const [starting, setStarting] = useState(true)

  useEffect(() => {
    let cancelled = false
    let html5Qrcode

    async function start() {
      try {
        // โหลดจาก CDN แบบ dynamic import — ไม่ต้องรอ npm install ก็ทดสอบได้ทันที
        // (ถ้าจะ build จริงแนะนำ `npm install html5-qrcode` แล้วเปลี่ยนมา import ตรงๆ)
        const { Html5Qrcode } = await import(
          /* @vite-ignore */ 'https://cdn.jsdelivr.net/npm/html5-qrcode@2.3.8/+esm'
        )

        if (cancelled) return

        html5Qrcode = new Html5Qrcode(SCANNER_ELEMENT_ID)
        scannerRef.current = html5Qrcode

        await html5Qrcode.start(
          { facingMode: 'environment' },
          { fps: 10, qrbox: { width: 260, height: 160 } },
          (decodedText) => {
            // สแกนเจอ — สั่นเตือน (ถ้าเครื่องรองรับ) แล้วส่งค่ากลับ
            if (navigator.vibrate) navigator.vibrate(120)
            onDetected(decodedText)
          },
          () => {
            // เฟรมที่ยังไม่เจอบาร์โค้ด — เงียบไว้ ไม่ต้องโชว์ error รัวๆ
          }
        )

        if (!cancelled) setStarting(false)
      } catch (err) {
        if (!cancelled) {
          setError(
            err?.message?.includes('Permission')
              ? 'กรุณาอนุญาตให้ใช้กล้องก่อนสแกน'
              : 'เปิดกล้องไม่สำเร็จ ลองใหม่อีกครั้ง หรือพิมพ์เลขเองแทน'
          )
          setStarting(false)
        }
      }
    }

    start()

    return () => {
      cancelled = true
      if (html5Qrcode) {
        html5Qrcode.stop().then(() => html5Qrcode.clear()).catch(() => {})
      }
    }
  }, [onDetected])

  return (
    <div className="wh-modal-overlay" onClick={onClose}>
      <div className="scanner-modal" onClick={(e) => e.stopPropagation()}>
        <h3 className="wh-modal-title">{title}</h3>

        <div id={SCANNER_ELEMENT_ID} className="scanner-viewport" />

        {starting && !error && <p className="wh-subtitle">กำลังเปิดกล้อง...</p>}
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}

        <div className="wh-modal-actions">
          <button className="wh-modal-cancel" onClick={onClose}>
            ปิด
          </button>
        </div>
      </div>
    </div>
  )
}