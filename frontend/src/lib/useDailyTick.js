// ─────────────────────────────────────────────────────────────────────────────
// useDailyTick — บังคับให้ component re-render เมื่อ "วันนี้" เปลี่ยน
//
// ป้ายสถานะ/เลข "เหลือ X วัน" คำนวณจาก new Date() ตอน render เท่านั้น ถ้าเปิด
// หน้าจอทิ้งไว้ข้ามเที่ยงคืน ตัวเลขจะค้างจนกว่าจะมีอะไรมากระตุ้น hook นี้แก้โดย:
//   1. ตั้ง timer ให้ยิงตอนเที่ยงคืนถัดไปพอดี แล้วตั้งใหม่ทุกวัน
//   2. ยิงเพิ่มเมื่อกลับมาโฟกัส/มองเห็นแท็บอีกครั้ง (เผื่อสลับแท็บทิ้งไว้หลายวัน)
//
// คืนค่าเป็น "คีย์ของวัน" (YYYY-MM-DD) ที่เปลี่ยนก็ต่อเมื่อข้ามวันจริง ๆ เอาไปใส่
// ใน dependency array ของ useMemo ที่คำนวณสถานะอายุได้ตรง ๆ
// ─────────────────────────────────────────────────────────────────────────────
import { useEffect, useState } from 'react'

// คีย์ของวันตามเวลาเครื่องผู้ใช้ (local) — ไม่ใช้ ISO/UTC เพราะเที่ยงคืนที่เรา
// สนใจคือเที่ยงคืนของผู้ใช้ ตรงกับ atMidnight ใน licenseExpiry.js
function dayKey(d = new Date()) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

// มิลลิวินาทีจากตอนนี้จนถึงเที่ยงคืนถัดไป (+1000ms กันพลาดขอบเขตวัน)
function msUntilNextMidnight() {
  const now = new Date()
  const next = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1, 0, 0, 0, 0)
  return next.getTime() - now.getTime() + 1000
}

export function useDailyTick() {
  const [today, setToday] = useState(() => dayKey())

  useEffect(() => {
    let timer

    // อัปเดต state เฉพาะเมื่อวันเปลี่ยนจริง เพื่อไม่ให้ re-render ฟรี ๆ
    const sync = () => setToday((prev) => (prev === dayKey() ? prev : dayKey()))

    // ตั้ง timer ไปเที่ยงคืนถัดไป พอถึงแล้ว sync + ตั้ง timer รอบใหม่
    const schedule = () => {
      clearTimeout(timer)
      timer = setTimeout(() => {
        sync()
        schedule()
      }, msUntilNextMidnight())
    }
    schedule()

    // กลับมาโฟกัส/มองเห็นแท็บ → เช็ควันทันที (เผื่อ timer ถูก throttle ตอน tab หลับ)
    const onWake = () => {
      if (document.visibilityState === 'visible') sync()
    }
    window.addEventListener('focus', onWake)
    document.addEventListener('visibilitychange', onWake)

    return () => {
      clearTimeout(timer)
      window.removeEventListener('focus', onWake)
      document.removeEventListener('visibilitychange', onWake)
    }
  }, [])

  return today
}
