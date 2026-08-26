import { useMemo, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import SelectField from '../components/Selectfield.jsx'
import { ColumnAliasPanel, CodeAliasPanel } from '../components/FormatTools.jsx'
import '../components/FormatTools.css'
import { ADMIN_NAV_ITEMS } from './AdminDashboardpage.jsx'
import { FORMAT_NAV_ITEMS } from './MasterDataPage.jsx'

const TARGET_COLUMNS = {
  planning: [
    'Line', 'LOT NO.', 'Machine', 'Product Spec 1', 'Product Spec 2', 'Domestic/Exp',
    'Assembly Status', 'Shipping Status', 'KCM Order', 'Country', 'Country Name', 'Brand',
    'Destination', 'IT device', 'IT Controller', 'IT Controller No', 'Swing Motor No',
    'Pump Assy HYD No', 'Motor Propel No', 'Control Valve No',
    'Front ATT', 'Engine start key', 'Note1', 'Note2', 'Note3',
  ],
  wh1: [
    'Warehouse', 'Order No', 'Work order', 'Parts No', 'Name',
    'Assembly Parts Number', 'Assembly Parts Name', 'Note', 'Final Color',
  ],
  wh2: ['Order', 'ORDER No.', 'Parts No', 'PARTS NAME', 'Quantity', 'LOCATION', 'Note'],
  engine: ['Machine No', 'History', 'ENGINE'],
  assembly: [
    'Machine No', 'Spec Code', 'Specification Detail', 'Country Name', 'IT device',
    'IT Controller No', 'Assembly_Parts_Number', 'Assembly_Parts_Name',
  ],
  import_license: [
    'ลำดับ', 'ตราอักษร',
    { value: 'รุ่น', label: 'แบบ/รุ่น' },
    'เลขใบอนุญาตนำเข้า', 'วันที่ออกใบอนุญาต', 'เลขอินวอยซ์นำเข้า', 'เลขใบขนสินค้าขาเข้า',
    { value: 'จำนวน', label: 'จำนวน (เครื่อง)' },
    'หมายเลขเครื่อง', 'หมายเลขการผลิต', 'หมายเหตุ', 'ส่งออกไปประเทศ',
  ],
  export_license: [
    'Item', "Date Ass'y", 'Machine No',
    { value: 'IT Controller Serial No.', label: 'IT Controller S/N' },
    'Country',
    { value: 'Invoice no.', label: 'Invoice' },
    'Export Entry', 'Import License', 'Export License',
    { value: 'Declaration date', label: 'วันที่นำออกใบอนุญาต' },
    'Remark',
  ],
  machine_spec: [
    'Machine No', 'KCM Order', 'Country Name', 'IT device', 'IT Controller', 'IT Controller S/N',
    'Engine', 'Engine History', 'Control valve', 'Motor Propel', 'Pump Assy HYD', 'HYD oil',
    'Boom', 'Arm', 'Shoe', 'Seat', 'Radio',
  ],
  wh_stock_mc: [
    { value: 'orderno', label: 'Order No' },
    { value: 'partsno', label: 'Parts No' },
    { value: 'name', label: 'Name' },
    { value: 'workorder', label: 'Work order' },
    { value: 'assemblypartsnumber', label: 'Assembly Parts Number' },
    { value: 'assemblypartsname', label: 'Assembly Parts Name' },
    { value: 'reservationno', label: 'Reservation No' },
    { value: 'finalcolor', label: 'Final Color' },
    { value: 'note', label: 'Note' },
  ],
  wh_stock_inv: [
    { value: 'pono', label: 'P.O.NO' },
    { value: 'partsno', label: 'Parts No' },
    { value: 'cno', label: 'C.No' },
    { value: 'description', label: 'Description' },
    { value: 'qty', label: 'Qty' },
    { value: 'sloc', label: 'Sloc' },
    { value: 'shelf', label: 'Shelf' },
  ],
  serial_list: [
    { value: 'IT Controller no.', label: 'IT Controller no. (หมายเลขเครื่อง)' },
    { value: 'หมายเลขการผลิต', label: 'IMEI (หมายเลขการผลิต)' },
    { value: 'Serial No', label: 'Serial No' },
    { value: 'Part No', label: 'Part No' },
    { value: 'Part Name', label: 'Part Name' },
    { value: 'Model', label: 'Model' },
    { value: 'เลขใบอนุญาตนำเข้า', label: 'เลขใบอนุญาตนำเข้า' },
    { value: 'เลขอินวอยซ์นำเข้า', label: 'เลขอินวอยซ์นำเข้า' },
  ],
}

const MASTER_DATA_BASE = ['Item No', 'Part Name', 'Model', 'Part No', 'Serial No']
const MASTER_DATA_TAIL = ['IMEI']
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
  // กลุ่ม "อื่น ๆ" (machine_spec, wh_stock_mc, wh_stock_inv, serial_list) ถูกเอาออกจากตัวเลือก
  // เพราะไม่ได้ใช้งาน — คอลัมน์ของ scope เหล่านี้ยังอยู่ใน TARGET_COLUMNS ด้านบน
  // ถ้าต้องการเปิดใช้อีกครั้ง ให้เพิ่มบรรทัด { scope, label, group: 'อื่น ๆ' } กลับมาที่นี่
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

        <div className="fmt-card">
          <label className="fmt-label" style={{ display: 'block', marginBottom: 8 }}>
            เลือกไฟล์ / งานที่ต้องการตั้งค่า
          </label>
          <div style={{ maxWidth: 440 }}>
            <SelectField value={scope} onChange={setScope} options={scopeOptions} />
          </div>
        </div>

        <div className="fmt-card">
          <h3 className="fmt-card-title">หัวคอลัมน์เปลี่ยนชื่อ / เพิ่มใหม่ / สลับตำแหน่ง</h3>
          <ColumnAliasPanel scope={scope} targetOptions={targetOptions} embedded />
        </div>

        <div className="fmt-card">
          <h3 className="fmt-card-title">Change Format Part</h3>
          <CodeAliasPanel componentType="it_controller" embedded />
        </div>
      </div>
    </AppShell>
  )
}