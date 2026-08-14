import Swal from 'sweetalert2'
import 'sweetalert2/dist/sweetalert2.min.css'
import { toastSuccess } from './toast.js'

// สีธีมหลักของแอป (teal) — ใช้กับปุ่มใน SweetAlert ให้เข้ากับหน้าเว็บ
// สีปุ่ม/ฟอนต์ของ popup ถูกกำหนดไว้ที่ src/theme.css (.swal2-*) แล้ว
// จึงไม่ต้องส่ง confirmButtonColor แบบ inline เข้ามาอีก

/**
 * เปิด SweetAlert popup สำหรับสแกน 1 ช่อง
 * - โฟกัสช่อง input อัตโนมัติ
 * - เครื่องสแกนพิมพ์รหัสเข้าช่องเอง แล้วส่ง Enter -> ปิด popup อัตโนมัติ
 * คืนค่า: string ที่สแกนได้ (trim แล้ว) หรือ null ถ้าผู้ใช้กดยกเลิก
 *
 * @param {object}   opts
 * @param {string}   opts.title        หัวข้อ popup
 * @param {string}  [opts.html]        คำอธิบาย/บริบท (HTML)
 * @param {string}  [opts.placeholder] ข้อความ placeholder ในช่อง input
 * @param {string}  [opts.confirmText] ข้อความปุ่มยืนยัน (เช่น 'ต่อไป' หรือ 'บันทึก')
 * @param {string}  [opts.cancelText]  ข้อความปุ่มยกเลิก (ใช้ 'ข้ามขั้นนี้' สำหรับขั้นที่ไม่บังคับ)
 * @param {(v:string)=>string|null} [opts.validate] ตรวจรูปแบบ คืน error string ถ้าไม่ผ่าน
 */
export async function scanStep({
  title,
  html = '',
  placeholder = 'รอรับสัญญาณจากเครื่องสแกน...',
  confirmText = 'ต่อไป',
  cancelText = 'ยกเลิก',
  validate,
}) {
  const res = await Swal.fire({
    title,
    html,
    input: 'text',
    inputPlaceholder: placeholder,
    inputAutoFocus: true,
    inputAttributes: {
      autocomplete: 'off',
      autocorrect: 'off',
      autocapitalize: 'off',
      spellcheck: 'false',
    },
    customClass: { popup: 'scan-popup', input: 'scan-popup-input' },
    confirmButtonText: confirmText,
    showCancelButton: true,
    cancelButtonText: cancelText,
    allowEnterKey: false, // จัดการ Enter เองด้านล่าง กันยิงซ้ำ
    inputValidator: (v) => {
      const val = (v || '').trim()
      if (!val) return 'ยังไม่มีค่าที่สแกน'
      if (validate) return validate(val) || undefined
      return undefined
    },
    didOpen: () => {
      const input = Swal.getInput()
      if (!input) return
      input.focus() // โฟกัสช่อง input อัตโนมัติ

      let confirmed = false
      const doConfirm = () => {
        if (confirmed) return
        if (!input.value.trim()) return
        confirmed = true
        Swal.clickConfirm()
      }

      // (1) เครื่องสแกนแบบ keyboard-wedge ส่ง Enter ปิดท้าย -> ยืนยันทันที
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault()
          doConfirm()
        }
      })

      // (2) เครื่องสแกนแบบ "วาง" (paste) ยิงรหัสมาทั้งก้อนทีเดียว -> ยืนยันอัตโนมัติ
      input.addEventListener('paste', () => {
        // รอให้ค่าถูกวางเข้าช่องก่อนแล้วค่อยยืนยัน
        setTimeout(doConfirm, 0)
      })

      // (3) ตรวจจับเครื่องสแกนจาก "ความเร็วการพิมพ์": สแกนเนอร์ยิงตัวอักษรรัวมาก
      // (หลายตัวใน < ~150ms) ซึ่งมือคนพิมพ์ไม่ทัน จึงแยกจากการพิมพ์เองบนมือถือ
      // ได้ชัด (รวมถึง autocomplete/คำแนะนำที่แทรกทีละหลายตัว — พวกนั้นไม่ได้มา
      // เป็น keydown รัวๆ) ถ้าเจอ burst แบบนี้ค่อยยืนยันให้เอง
      let burstStart = 0
      let burstCount = 0
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key.length !== 1) return // นับเฉพาะตัวอักษรจริง
        const now = Date.now()
        if (now - burstStart > 150) {
          burstStart = now
          burstCount = 0
        }
        burstCount++
        if (burstCount >= 6) {
          setTimeout(doConfirm, 60) // รอตัวอักษรสุดท้ายเข้าช่องก่อน
        }
      })
    },
  })

  if (res.isConfirmed && res.value) return res.value.trim()
  return null // ยกเลิก / ปิด
}

/**
 * ให้ผู้ใช้เลือกชนิดพาร์ท (ใช้เป็น fallback เมื่อบาร์โค้ดที่ยิงมาระบุพาร์ทไม่ได้)
 * @param {{title:string, html?:string, options:{value:string,label:string}[]}} opts
 * @returns {Promise<string|null>} value ที่เลือก หรือ null ถ้ายกเลิก
 */
export async function scanSelect({ title, html = '', options }) {
  const inputOptions = {}
  options.forEach((o) => {
    inputOptions[o.value] = o.label
  })
  const res = await Swal.fire({
    title,
    html,
    input: 'select',
    inputOptions,
    customClass: { popup: 'scan-popup' },
    inputPlaceholder: 'เลือกชนิดพาร์ท',
    confirmButtonText: 'ต่อไป',
    showCancelButton: true,
    cancelButtonText: 'ยกเลิก',
  })
  return res.isConfirmed ? res.value : null
}

/** แสดง loading ระหว่างบันทึก (ปิดด้วย Swal.close() หรือ popup ตัวถัดไป) */
export function scanLoading(title = 'กำลังบันทึก...') {
  Swal.fire({
    title,
    allowOutsideClick: false,
    allowEscapeKey: false,
    didOpen: () => Swal.showLoading(),
  })
}

/** ปิด popup ที่เปิดอยู่ (เช่น หลังโหลดเสร็จ) */
export function scanClose() {
  Swal.close()
}

/**
 * ปิด popup ปัจจุบันแล้ว "รอจนปิดสนิท" ก่อน resolve
 *
 * ใช้ก่อนจะเปิด popup ตัวใหม่ที่มี didOpen (เช่น เปิดกล้องถ่ายรูป) — กันสองปัญหา
 * ของ SweetAlert2 พร้อมกัน:
 *   1) ถ้าเรียก Swal.fire() ทับ popup เดิมที่ยัง "เปิดอยู่" SweetAlert จะไม่รัน
 *      didOpen ของตัวใหม่ (เพราะถือว่า popup เปิดอยู่แล้ว) => กล้องไม่เริ่มทำงาน /
 *      หน้าถ่ายรูปไม่ขึ้น
 *   2) ถ้าเรียก Swal.close() แล้ว fire() ต่อทันที อนิเมชันปิดที่ค้างอยู่จะวิ่งไปลบ
 *      popup ตัวใหม่ => หน้าถ่ายรูป "หาย"
 * วิธีที่ชัวร์คือรอให้ตัวเดิมปิดจนจบ animation แล้วค่อยเปิดตัวใหม่เป็น popup สดใหม่
 */
export function scanCloseWait() {
  return new Promise((resolve) => {
    const popup = Swal.getPopup()
    if (!popup) {
      resolve()
      return
    }
    let done = false
    const finish = () => {
      if (done) return
      done = true
      resolve()
    }
    popup.addEventListener('animationend', finish, { once: true })
    popup.addEventListener('transitionend', finish, { once: true })
    Swal.close()
    // fallback: บาง config ไม่มี animation ปิด — กันค้างไม่ให้ resolve ไม่มา
    setTimeout(finish, 320)
  })
}

/** toast แจ้งเตือนสำเร็จ มุมขวาบน — ใช้ตัวเดียวกับที่อื่นทั้งระบบ */
export function scanSuccessToast(title) {
  return toastSuccess(title)
}

/** popup แจ้ง error พร้อมปุ่มลองใหม่ */
export function scanErrorAlert(text) {
  return Swal.fire({
    icon: 'error',
    title: 'เกิดข้อผิดพลาด',
    text,
    confirmButtonText: 'ตกลง',
  })
}

/**
 * เปิดกล้อง (กล้องหลังบนมือถือ) ให้ถ่ายรูปป้ายเพื่อยืนยัน หลังจากสแกน
 * P/N + S/N เสร็จ — ใช้คู่กับ uploadPartCheckPhoto ฝั่ง api/partcheck.js
 *
 * เป็นขั้นตอน "บังคับ" ต่อจากสแกน P/N + S/N เสมอ (flow เดียว ไม่มีปุ่มข้าม)
 * คืนค่า Blob รูป (JPEG) เมื่อถ่ายสำเร็จ, หรือ null ถ้าเปิดกล้องไม่สำเร็จจริงๆ
 * (เช่น ไม่มีกล้อง/ไม่ได้อนุญาต permission/ไม่ใช่ secure context)
 *
 * @param {object} opts
 * @param {string} opts.title หัวข้อ popup
 * @param {string} [opts.html] คำอธิบาย/บริบท (HTML) — เช่น โชว์ P/N, S/N ที่สแกนไว้
 */
export async function scanPhotoCapture({ title, html = '' }) {
  let stream = null

  const res = await Swal.fire({
    title,
    html: `
      ${html}
      <video id="scan-photo-video" autoplay playsinline muted class="scan-photo-video"></video>
      <canvas id="scan-photo-canvas" style="display:none;"></canvas>
    `,
    customClass: { popup: 'scan-popup' },
    confirmButtonText: 'ถ่ายรูป',
    showCancelButton: true,
    cancelButtonText: 'ปิด',
    allowOutsideClick: false,
    allowEscapeKey: true,
    didOpen: async () => {
      const video = document.getElementById('scan-photo-video')
      try {
        // กล้อง (getUserMedia) ใช้ได้เฉพาะ secure context: HTTPS หรือ http://localhost
        // ถ้าเปิดผ่าน http://<ip> ธรรมดา navigator.mediaDevices จะเป็น undefined
        if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
          Swal.showValidationMessage(
            'เปิดกล้องไม่ได้: ต้องเข้าเว็บผ่าน https (เช่น https://' +
              location.host +
              ') บนมือถือ กล้องถึงจะทำงาน'
          )
          return
        }
        stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: 'environment' },
          audio: false,
        })
        if (video) video.srcObject = stream
      } catch (err) {
        Swal.showValidationMessage('เปิดกล้องไม่สำเร็จ: ' + (err.message || err))
        // เปิดกล้องไม่ได้จริงๆ (ไม่มีกล้อง/ไม่ได้อนุญาต/ไม่ใช่ secure context) —
        // ให้ทางออกฉุกเฉินเฉพาะกรณีนี้ ปกติ flow นี้ไม่มีปุ่มข้าม
        Swal.update({ showCancelButton: true, cancelButtonText: 'ปิด' })
      }
    },
    preConfirm: () => {
      const video = document.getElementById('scan-photo-video')
      const canvas = document.getElementById('scan-photo-canvas')
      if (!video || !canvas || !video.videoWidth) {
        Swal.showValidationMessage('กล้องยังไม่พร้อม กรุณารอสักครู่แล้วลองใหม่')
        return false
      }
      const vw = video.videoWidth
      const vh = video.videoHeight
      // crop กลางภาพให้เป็นสี่เหลี่ยมจัตุรัส (ตรงกับกรอบพรีวิวที่เป็น 1:1)
      const side = Math.min(vw, vh)
      const sx = (vw - side) / 2
      const sy = (vh - side) / 2
      canvas.width = side
      canvas.height = side
      canvas.getContext('2d').drawImage(video, sx, sy, side, side, 0, 0, side, side)
      return new Promise((resolve) => {
        canvas.toBlob((blob) => resolve(blob), 'image/jpeg', 0.9)
      })
    },
    willClose: () => {
      // ปิดกล้องเสมอเมื่อ popup นี้ปิดลง (ถ่ายรูปสำเร็จ) — กันไฟกล้องค้างเปิด
      if (stream) stream.getTracks().forEach((t) => t.stop())
    },
  })

  if (res.isConfirmed && res.value) return res.value
  return null
}