import Swal from 'sweetalert2'
import 'sweetalert2/dist/sweetalert2.min.css'

/**
 * toast มุมขวาบน ใช้ร่วมกันทั้งระบบ
 *
 * - เลื่อนเมาส์ไปวางบน toast แล้วตัวจับเวลาจะหยุด เอาเมาส์ออกแล้วนับต่อ
 *   (เผื่อข้อความยาวหรืออ่านไม่ทัน)
 * - หน้าตาปุ่ม/ฟอนต์/มุมโค้ง ถูกกำหนดไว้ที่ theme.css (.swal2-*) แล้ว
 */
export const Toast = Swal.mixin({
  toast: true,
  position: 'top-end',
  showConfirmButton: false,
  timer: 3000,
  timerProgressBar: true,
  didOpen: (toast) => {
    toast.onmouseenter = Swal.stopTimer
    toast.onmouseleave = Swal.resumeTimer
  },
})

/** แจ้งว่าทำสำเร็จ เช่น เข้าสู่ระบบ / ลบ / อัปโหลด */
export function toastSuccess(title) {
  return Toast.fire({ icon: 'success', title })
}

/** แจ้งว่าไม่สำเร็จ — ใช้กับงานที่ไม่ต้องให้ผู้ใช้กดรับทราบ */
export function toastError(title) {
  return Toast.fire({ icon: 'error', title })
}

/**
 * กล่องยืนยันก่อนทำสิ่งที่กู้คืนไม่ได้ (แทน window.confirm ของเบราว์เซอร์)
 * คืนค่า true เมื่อผู้ใช้กดยืนยัน
 */
export async function confirmDelete({
  title = 'ยืนยันการลบ',
  text,
  confirmText = 'ลบ',
}) {
  const res = await Swal.fire({
    icon: 'warning',
    title,
    text,
    showCancelButton: true,
    focusCancel: true,
    confirmButtonText: confirmText,
    cancelButtonText: 'ยกเลิก',
    customClass: { confirmButton: 'swal2-confirm-danger' },
  })
  return res.isConfirmed
}
