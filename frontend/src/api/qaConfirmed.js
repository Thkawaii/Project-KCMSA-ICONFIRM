import { apiFetch } from './client.js'

// ดึงตารางสรุป QA — รวมข้อมูลจาก WH (Part Confirmation) + MFG (Assembly) + Master Data
// เฉพาะเครื่องที่ครบเงื่อนไข: MFG สแกนแล้ว MATCHED และ WH ยืนยันตรงกับใบอนุญาตนำเข้า
// คืน array ของแถว { partName, model, machineNo, partNo, serialNo, itControllerNo,
//                    imei, licenseNo, invoiceNo, matchStatus, matchMessage, photoURL, status }
export function getQAConfirmedTable() {
  return apiFetch('/qa/confirmed')
}
