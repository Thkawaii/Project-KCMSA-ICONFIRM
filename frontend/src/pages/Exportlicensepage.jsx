import AppShell from '../components/AppShell.jsx'
import { WH_NAV_ITEMS, WHExportLicensePanel } from './Importlicensepage.jsx'

// หน้า Export License — แยกออกมาเป็นเมนูหลักของตัวเอง อยู่ถัดจาก Import License
// เนื้อหาเดิมคือแท็บ "Export License" ในหน้า Import License (ย้ายออกมาทั้งหมด)
export default function ExportLicensePage() {
  return (
    <AppShell navItems={WH_NAV_ITEMS} roleLabel="Warehouse">
      <div className="wh-heading-row">
        <div>
          <h2 className="wh-title">Export License</h2>
        </div>
      </div>

      <WHExportLicensePanel />
    </AppShell>
  )
}
