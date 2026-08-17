import { useMemo, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import SelectField from '../components/Selectfield.jsx'
import { ColumnAliasPanel, CodeAliasPanel } from '../components/FormatTools.jsx'
import '../components/FormatTools.css'
import { ADMIN_NAV_ITEMS } from './AdminDashboardpage.jsx'
import { FORMAT_NAV_ITEMS } from './MasterDataPage.jsx'

// ─────────────────────────────────────────────────────────────────────────────
// หน้า "ตั้งค่า Format" (Setting) — เรียบง่าย ใช้งานง่าย รองรับมือถือ
//
//  • เลือก "ไฟล์/งาน" ที่จะตั้งค่า (เฉพาะไฟล์ข้อมูล + ใบอนุญาต)
//  • Column Alias: จับคู่หัวคอลัมน์ใหม่ → คอลัมน์เดิม (เลือกจาก dropdown)
//  • Change Format Part: จับคู่ค่ารหัส (Machine No./P/N/S/N) ใหม่ → เดิม
// ─────────────────────────────────────────────────────────────────────────────

// TARGET_COLUMNS = "ข้อมูลเดิม (คอลัมน์มาตรฐาน)" ที่ถูกต้องของแต่ละ scope (dropdown)
//   • กลุ่มข้อมูล (planning/wh1/wh2/engine/assembly) ต้องตรงกับ "Label" เป๊ะ ๆ
//   • กลุ่มใบอนุญาต ระบบ normalize ก่อนเทียบ จึงใช้ชื่อหัวมาตรฐานได้
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
  // Import License — option ตรงกับหัวคอลัมน์ที่แสดงในตาราง (value = คีย์ที่ระบบรู้จัก)
  import_license: [
    'ลำดับ', 'ตราอักษร',
    { value: 'รุ่น', label: 'แบบ/รุ่น' },
    'เลขใบอนุญาตนำเข้า', 'วันที่ออกใบอนุญาต', 'เลขอินวอยซ์นำเข้า', 'เลขใบขนสินค้าขาเข้า',
    { value: 'จำนวน', label: 'จำนวน (เครื่อง)' },
    'หมายเลขเครื่อง', 'หมายเลขการผลิต', 'หมายเหตุ', 'ส่งออกไปประเทศ',
  ],
  // Export License — option ตรงกับหัวคอลัมน์ที่แสดงในตาราง (value = คีย์ที่ระบบรู้จัก)
  export_license: [
    'Item', "Date Ass'y", 'Machine No',
    { value: 'IT Controller Serial No.', label: 'IT Controller S/N' },
    'Country',
    { value: 'Invoice no.', label: 'Invoice' },
    'Export Entry', 'Import License', 'Export License',
    { value: 'Declaration date', label: 'วันที่นำออกใบอนุญาต' },
    'Remark',
  ],
}

// คอลัมน์มาตรฐานของ "รายการ (Master Data)" — ทุกชนิดใช้หัวคอลัมน์เดียวกัน แต่ช่อง
// "เลขประจำเครื่อง" (ITControllerNo) แสดงชื่อ option ต่างกันตามชนิด (value เดิม =
// "IT Controller no." ที่ระบบรู้จัก จึงยัง map ถูกเสมอ) — ไม่มี Spec Code / Connectivity
const MASTER_DATA_BASE = ['Item No', 'Part Name', 'Model', 'Part No', 'Serial No']
const MASTER_DATA_TAIL = ['IMEI']
// ชื่อ option ของช่องเลขประจำเครื่อง ตามแต่ละ scope
const ITC_LABEL_BY_SCOPE = {
  master_data: 'No.',
  'master_data:it_controller': 'IT Controller no.',
  'master_data:swing_motor': 'Swing Motor No.',
  'master_data:pump_assy_hyd': 'Pump Assy HYD NO.',
  'master_data:motor_propel': 'Motor Propel NO.',
  'master_data:control_valve': 'Control Valve NO.',
}
Object.keys(ITC_LABEL_BY_SCOPE).forEach((sc) => {
  TARGET_COLUMNS[sc] = [
    ...MASTER_DATA_BASE,
    { value: 'IT Controller no.', label: ITC_LABEL_BY_SCOPE[sc] },
    ...MASTER_DATA_TAIL,
  ]
})

// รายการ "ไฟล์/งาน" — ไฟล์ข้อมูล + รายการทะเบียน (ราย component) + ใบอนุญาต
const SCOPES = [
  { scope: 'planning', label: 'Planning', group: 'ข้อมูล' },
  { scope: 'wh1', label: 'Warehouse 1', group: 'ข้อมูล' },
  { scope: 'wh2', label: 'Warehouse 2', group: 'ข้อมูล' },
  { scope: 'engine', label: 'Engine', group: 'ข้อมูล' },
  { scope: 'assembly', label: 'Assembly', group: 'ข้อมูล' },
  { scope: 'master_data', label: 'รายการ ทุกชนิด', group: 'ข้อมูล' },
  { scope: 'master_data:it_controller', label: 'รายการ IT Controller', group: 'ข้อมูล' },
  { scope: 'master_data:swing_motor', label: 'รายการ Swing Motor', group: 'ข้อมูล' },
  { scope: 'master_data:pump_assy_hyd', label: 'รายการ Pump Assy HYD', group: 'ข้อมูล' },
  { scope: 'master_data:motor_propel', label: 'รายการ Motor Propel', group: 'ข้อมูล' },
  { scope: 'master_data:control_valve', label: 'รายการ Control Valve', group: 'ข้อมูล' },
  { scope: 'import_license', label: 'Import License', group: 'ใบอนุญาต' },
  { scope: 'export_license', label: 'Export License', group: 'ใบอนุญาต' },
]

export default function FormatSettingsPage() {
  const isAdmin = (localStorage.getItem('iconfirm_role') || '').toUpperCase() === 'ADMIN'
  const navItems = isAdmin ? ADMIN_NAV_ITEMS : FORMAT_NAV_ITEMS
  const shellRoleLabel = isAdmin ? 'Admin' : 'Upload View'

  const [scope, setScope] = useState('planning')
  const active = SCOPES.find((s) => s.scope === scope) || SCOPES[0]
  const targetOptions = TARGET_COLUMNS[scope] || []

  const scopeOptions = useMemo(
    () => SCOPES.map((s) => ({ value: s.scope, label: `${s.group} › ${s.label}` })),
    [],
  )

  return (
    <AppShell navItems={navItems} roleLabel={shellRoleLabel}>
      <div className="fmt-page">
        <div className="fmt-page-head">
          <h2 style={{ fontWeight: 700, fontSize: 22, color: 'var(--color-brand-ink, #06312f)', margin: 0 }}>
            ตั้งค่า Format
          </h2>
          <span style={{ fontSize: 12.5, color: '#94a3b8' }}>
            มีผลกับการอัปโหลดครั้งถัดไป (ไม่ย้อนหลังข้อมูลเดิม)
          </span>
        </div>

        {/* เลือกไฟล์/งาน */}
        <div className="fmt-card">
          <label className="fmt-label" style={{ display: 'block', marginBottom: 8 }}>
            เลือกไฟล์ / งานที่ต้องการตั้งค่า
          </label>
          <div style={{ maxWidth: 440 }}>
            <SelectField value={scope} onChange={setScope} options={scopeOptions} />
          </div>
        </div>

        {/* Column Alias */}
        <div className="fmt-card">
          <h3 className="fmt-card-title">หัวคอลัมน์เปลี่ยนชื่อ / เพิ่มใหม่ / สลับตำแหน่ง</h3>
          <ColumnAliasPanel scope={scope} targetOptions={targetOptions} embedded />
        </div>

        {/* Change Format Part (Code Alias) */}
        <div className="fmt-card">
          <h3 className="fmt-card-title">Change Format Part</h3>
          <CodeAliasPanel componentType="it_controller" embedded />
        </div>
      </div>
    </AppShell>
  )
}
