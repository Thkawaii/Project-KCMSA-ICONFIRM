import { useMemo, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import SelectField from '../components/Selectfield.jsx'
import { ColumnAliasPanel, CodeAliasPanel } from '../components/FormatTools.jsx'
import { ADMIN_NAV_ITEMS } from './AdminDashboardpage.jsx'
import { FORMAT_NAV_ITEMS } from './MasterDataPage.jsx'

// ─────────────────────────────────────────────────────────────────────────────
// หน้า "ตั้งค่า Format" (Setting) — ปรับให้เรียบง่าย ใช้งานง่าย
//
//  • เลือก "ไฟล์/งาน" ที่จะตั้งค่า → ตั้งค่า Column Alias ได้ทันที
//  • ช่อง "คอลัมน์มาตรฐาน" เปลี่ยนเป็น dropdown ให้เลือกจากรายการที่ถูกต้อง
//    (เดิมพิมพ์เองแล้วมักไม่ตรงชื่อคอลัมน์เป๊ะ ๆ จึง "ตั้งแล้วไม่เปลี่ยน")
//  • ลบคำอธิบายยาว ๆ ที่รกออก เหลือหมายเหตุสั้น ๆ ที่จำเป็น
// ─────────────────────────────────────────────────────────────────────────────

// TARGET_COLUMNS = "คอลัมน์มาตรฐาน" ที่ถูกต้องของแต่ละ scope (ให้เลือกจาก dropdown)
//   • กลุ่มอัปโหลด (planning/wh1/wh2/engine/assembly) ต้องตรงกับ "Label" เป๊ะ ๆ
//   • กลุ่มทะเบียน/ใบอนุญาต ระบบจะ normalize ก่อนเทียบ จึงใช้ชื่อหัวมาตรฐานได้
const TARGET_COLUMNS = {
  planning: [
    'Line', 'LOT NO.', 'Machine', 'Product Spec 1', 'Product Spec 2', 'Domestic/Exp',
    'Assembly Status', 'Shipping Status', 'KCM Order', 'Country', 'Country Name', 'Brand',
    'Destination', 'IT device', 'Front ATT', 'Engine start key', 'Note1', 'Note2', 'Note3',
  ],
  wh1: [
    'Warehouse', 'Order No', 'Work order', 'Parts No', 'Name',
    'Assembly Parts Number', 'Assembly Parts Name', 'Note', 'Final Color',
  ],
  wh2: ['Order', 'ORDER No.', 'Parts No', 'PARTS NAME', 'Quantity', 'LOCATION', 'Note'],
  engine: ['Machine No', 'History', 'ENGINE'],
  assembly: [
    'Machine No', 'Spec Code', 'Specification Detail', 'Country Name', 'IT device',
    'IT Controller', 'Assembly_Parts_Number', 'Assembly_Parts_Name',
  ],
  master_data: [
    'Item No', 'Part Name', 'Model', 'Part No', 'Serial No',
    'IT Controller no.', 'IMEI', 'Spec Code', 'Connectivity Type',
  ],
  machine_spec: [
    'Machine No', 'Spec(1)', 'Spec(2)', 'KCM Order', 'Country Name', 'IT device',
    'IT Controller', 'IT Controller S/N', 'Engine', 'CW no', 'CW name', 'Shoe', 'Seat',
  ],
  serial_list: [
    'IT Controller no.', 'IMEI', 'Part Name', 'Model', 'Brand', 'Part No', 'Serial No',
    'Import License No', 'Invoice No', 'Declaration No', 'Qty', 'Issue Date', 'Connectivity Type',
  ],
  import_license: [
    'ลำดับ', 'ตราอักษร', 'รุ่น', 'เลขใบอนุญาตนำเข้า', 'เลขอินวอยซ์นำเข้า',
    'เลขใบขนสินค้าขาเข้า', 'จำนวน', 'หมายเลขเครื่อง', 'หมายเลขการผลิต', 'หมายเหตุ',
    'ประเทศ', 'วันที่ออกใบอนุญาต',
  ],
  export_license: [
    'Item', "Date Ass'y", 'Machine No', 'IT Controller Serial No.', 'Serial Number',
    'Invoice no.', 'Invoice date', 'Export Entry', 'Import License', 'Export License',
    'Expire date', 'ใบขน (Date)',
  ],
}

// รายการ "ไฟล์/งาน" ทั้งหมด แยกเป็น 3 หมวดให้เลือกง่าย
const SCOPES = [
  { scope: 'planning', label: 'Planning', group: 'ไฟล์อัปโหลดประจำวัน' },
  { scope: 'wh1', label: 'Warehouse 1', group: 'ไฟล์อัปโหลดประจำวัน' },
  { scope: 'wh2', label: 'Warehouse 2', group: 'ไฟล์อัปโหลดประจำวัน' },
  { scope: 'engine', label: 'Engine', group: 'ไฟล์อัปโหลดประจำวัน' },
  { scope: 'assembly', label: 'Assembly', group: 'ไฟล์อัปโหลดประจำวัน' },
  { scope: 'master_data', label: 'Master Data (ทะเบียนกลาง)', group: 'ทะเบียนกลาง' },
  { scope: 'machine_spec', label: 'Machine Spec', group: 'ทะเบียนกลาง' },
  { scope: 'serial_list', label: 'Serial List (IT Controller)', group: 'ทะเบียนกลาง' },
  { scope: 'import_license', label: 'Import License', group: 'ใบอนุญาต' },
  { scope: 'export_license', label: 'Export License', group: 'ใบอนุญาต' },
]

const GROUPS = ['ไฟล์อัปโหลดประจำวัน', 'ทะเบียนกลาง', 'ใบอนุญาต']

// การ์ดครอบ section — โทนเรียบ สีขาว ขอบบาง
const cardStyle = {
  border: '1px solid #e2e8f0',
  borderRadius: 14,
  background: '#fff',
  padding: 18,
  marginBottom: 16,
}

export default function FormatSettingsPage() {
  const isAdmin = (localStorage.getItem('iconfirm_role') || '').toUpperCase() === 'ADMIN'
  const navItems = isAdmin ? ADMIN_NAV_ITEMS : FORMAT_NAV_ITEMS
  const shellRoleLabel = isAdmin ? 'Admin' : 'Upload View'

  const [scope, setScope] = useState('planning')
  const active = SCOPES.find((s) => s.scope === scope) || SCOPES[0]
  const targetOptions = TARGET_COLUMNS[scope] || []

  // จัดตัวเลือก dropdown ให้จัดกลุ่มด้วยหัวข้อ (แสดงเป็น label นำหน้าแต่ละกลุ่ม)
  const scopeOptions = useMemo(
    () => SCOPES.map((s) => ({ value: s.scope, label: `${s.group} › ${s.label}` })),
    [],
  )

  return (
    <AppShell navItems={navItems} roleLabel={shellRoleLabel}>
      <div style={{ padding: 16, maxWidth: 900 }}>
        <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', flexWrap: 'wrap', gap: 8, marginBottom: 14 }}>
          <h2 style={{ fontWeight: 700, fontSize: 22, color: 'var(--color-brand-ink, #06312f)', margin: 0 }}>
            ตั้งค่า Format
          </h2>
          <span style={{ fontSize: 12.5, color: '#94a3b8' }}>
            มีผลกับการอัปโหลดครั้งถัดไป (ไม่ย้อนหลังข้อมูลเดิม)
          </span>
        </div>

        {/* เลือกไฟล์/งาน */}
        <div style={{ ...cardStyle, paddingBottom: 14 }}>
          <label style={{ fontSize: 13, fontWeight: 600, color: '#334155', display: 'block', marginBottom: 8 }}>
            เลือกไฟล์ / งานที่ต้องการตั้งค่า
          </label>
          <div style={{ maxWidth: 420 }}>
            <SelectField value={scope} onChange={setScope} options={scopeOptions} />
          </div>
        </div>

        {/* Column Alias */}
        <div style={cardStyle}>
          <h3 style={{ fontWeight: 700, fontSize: 15, margin: '0 0 4px' }}>
            หัวคอลัมน์เปลี่ยนชื่อ / เพิ่มใหม่ / สลับตำแหน่ง
          </h3>
          <p style={{ fontSize: 12.5, color: '#64748b', margin: '0 0 12px' }}>
            พิมพ์ชื่อหัวคอลัมน์ที่ไฟล์เขียนมาจริง แล้วเลือกว่าให้ลงคอลัมน์มาตรฐานตัวไหนของ{' '}
            <b>{active.label}</b>
          </p>
          <ColumnAliasPanel scope={scope} targetOptions={targetOptions} embedded />
        </div>

        {/* Code Alias — เฉพาะ IT Controller (ค่า P/N · S/N · Machine No.) */}
        <div style={cardStyle}>
          <h3 style={{ fontWeight: 700, fontSize: 15, margin: '0 0 4px' }}>
            รหัส P/N · S/N · Machine No. หน้างานเปลี่ยนรูปแบบ
          </h3>
          <p style={{ fontSize: 12.5, color: '#64748b', margin: '0 0 12px' }}>
            ผูกค่าที่หน้างานยิงมาแบบใหม่ ให้ชี้กลับไปยังค่ามาตรฐานในทะเบียนกลาง
          </p>
          <CodeAliasPanel componentType="it_controller" embedded />
        </div>
      </div>
    </AppShell>
  )
}
