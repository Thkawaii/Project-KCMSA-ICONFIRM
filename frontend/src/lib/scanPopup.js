import Swal from 'sweetalert2'
import 'sweetalert2/dist/sweetalert2.min.css'

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
      // เครื่องสแกนส่ง Enter มา -> ยืนยันทันที (popup ปิดเอง)
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault()
          if (input.value.trim()) Swal.clickConfirm()
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

/** toast แจ้งเตือนสำเร็จ มุมขวาบน */
export function scanSuccessToast(title) {
  return Swal.fire({
    toast: true,
    position: 'top-end',
    icon: 'success',
    title,
    timer: 2500,
    showConfirmButton: false,
    timerProgressBar: true,
  })
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