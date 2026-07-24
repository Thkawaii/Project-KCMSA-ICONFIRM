/**
 * ไอคอนทั้งระบบมาจาก Heroicons (ชุดไอคอนอย่างเป็นทางการของทีม Tailwind)
 * ที่ไฟล์นี้ที่เดียว — ห้ามใช้ emoji หรือวาด <svg> เองในหน้าอื่นอีก
 *
 * เหตุผลที่รวมไว้ที่เดียว:
 *   1. emoji หน้าตาไม่เหมือนกันในแต่ละ OS (Windows / iOS / Android คนละชุด)
 *      และปรับสี/ขนาดตามข้อความรอบข้างไม่ได้ ส่วน Heroicons เป็น SVG ที่
 *      ใช้ currentColor จึงรับสีจากคลาส Tailwind ได้ตรงๆ
 *   2. ชนิดพาร์ท 5 อย่าง (IT Controller / Control Valve / ...) ถูกประกาศซ้ำ
 *      อยู่ใน 4 หน้า ถ้าใครเพิ่มชนิดใหม่จะได้แก้ไอคอนที่นี่จุดเดียว
 *
 * ขนาดที่ใช้: ใส่ผ่านคลาส Tailwind ตอนเรียก เช่น <TagIcon className="size-5" />
 */

import {
  ArrowDownTrayIcon,
  ArrowPathIcon,
  ArrowRightStartOnRectangleIcon,
  ArrowUpTrayIcon,
  ArrowsRightLeftIcon,
  BeakerIcon,
  CameraIcon,
  CheckCircleIcon,
  CheckIcon,
  ClipboardDocumentCheckIcon,
  ClockIcon,
  CloudArrowUpIcon,
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  Cog6ToothIcon,
  CpuChipIcon,
  CubeIcon,
  DocumentTextIcon,
  ExclamationTriangleIcon,
  EyeIcon,
  EyeSlashIcon,
  FunnelIcon,
  LockClosedIcon,
  MinusIcon,
  PencilSquareIcon,
  QrCodeIcon,
  ReceiptPercentIcon,
  RectangleStackIcon,
  ShieldCheckIcon,
  Squares2X2Icon,
  TagIcon,
  TruckIcon,
  WrenchScrewdriverIcon,
  XCircleIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline'

export {
  ArrowDownTrayIcon,
  ArrowPathIcon,
  ArrowRightStartOnRectangleIcon,
  ArrowUpTrayIcon,
  ArrowsRightLeftIcon,
  CameraIcon,
  CheckCircleIcon,
  CheckIcon,
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ClipboardDocumentCheckIcon,
  ClockIcon,
  CloudArrowUpIcon,
  CpuChipIcon,
  CubeIcon,
  DocumentTextIcon,
  ExclamationTriangleIcon,
  EyeIcon,
  EyeSlashIcon,
  FunnelIcon,
  LockClosedIcon,
  MinusIcon,
  PencilSquareIcon,
  QrCodeIcon,
  ReceiptPercentIcon,
  RectangleStackIcon,
  ShieldCheckIcon,
  Squares2X2Icon,
  TagIcon,
  XCircleIcon,
  XMarkIcon,
}

/**
 * ไอคอนประจำชนิดพาร์ท — key ตรงกับ component_type ที่ backend ใช้
 * (BeakerIcon แทน Pump Assy HYD เพราะ Heroicons ไม่มีไอคอนหยดน้ำ/ปั๊ม
 *  เลือกตัวที่สื่อถึง "ของเหลว" ใกล้เคียงที่สุด)
 */
export const PART_ICONS = {
  it_controller: CpuChipIcon,
  control_valve: WrenchScrewdriverIcon,
  swing_motor: Cog6ToothIcon,
  motor_propel: TruckIcon,
  pump_assy_hyd: BeakerIcon,
  machine: TruckIcon,
}

/** ใช้กับหน้า Part Confirmation ที่อ้างพาร์ทด้วยรหัสย่อแทน component_type */
export const PART_ICONS_BY_CODE = {
  MC: TruckIcon,
  ITC: CpuChipIcon,
  CV: WrenchScrewdriverIcon,
  SM: Cog6ToothIcon,
  MP: TruckIcon,
  PH: BeakerIcon,
}
