import { useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import { ColumnAliasPanel, CodeAliasPanel } from '../components/FormatTools.jsx'
import { ADMIN_NAV_ITEMS } from './AdminDashboardpage.jsx'
import { FORMAT_NAV_ITEMS } from './MasterDataPage.jsx'

// ─────────────────────────────────────────────────────────────────────────────
// หน้า "ตั้งค่า Format" — ให้หน้างาน/แอดมินรองรับการเปลี่ยน format ของไฟล์อัปโหลด
// และรหัส P/N / S/N / Machine No. ได้เองตอนรัน โดยไม่ต้องแก้โค้ด/รีดีพลอย
//
//  • Column Alias  = หัวคอลัมน์ในไฟล์เปลี่ยนชื่อ/เพิ่มใหม่/สลับตำแหน่ง → คอลัมน์มาตรฐาน
//    (ผูกกับ scope ของแต่ละ importer — backend รองรับทุก scope ด้านล่างแล้ว)
//  • Code Alias    = ค่า P/N / S/N / Machine No. รูปแบบใหม่หน้างาน → ค่ามาตรฐานในทะเบียน
// ─────────────────────────────────────────────────────────────────────────────

// scope = ชื่อ importer ฝั่ง backend (ต้องตรงกับที่ controller เรียก loadColumnAlias*)
const COLUMN_SCOPES = [
  { scope: 'planning', label: 'Planning', hint: 'Machine, Product Spec 1, KCM Order …' },
  { scope: 'wh1', label: 'Warehouse 1', hint: 'Parts No, Order No …' },
  { scope: 'wh2', label: 'Warehouse 2', hint: 'Parts No, Order No …' },
  { scope: 'engine', label: 'Engine', hint: 'Machine No, History, ENGINE' },
  { scope: 'master_data', label: 'Master Data (ทะเบียนกลาง)', hint: 'Part No, Serial No, IT Controller No, IMEI, Type' },
  { scope: 'machine_spec', label: 'Machine Spec', hint: 'Machine No, IT Controller S/N, CW no, Engine …' },
  { scope: 'serial_list', label: 'Serial List (IT Controller)', hint: 'IT Controller No, IMEI, Serial No, Part No …' },
  { scope: 'import_license', label: 'Import License', hint: 'Import License No, Invoice No, IT Controller No …' },
  { scope: 'export_license', label: 'Export License', hint: 'Export License, Serial Number, Machine No …' },
]

export default function FormatSettingsPage() {
  const isAdmin = (localStorage.getItem('iconfirm_role') || '').toUpperCase() === 'ADMIN'
  const navItems = isAdmin ? ADMIN_NAV_ITEMS : FORMAT_NAV_ITEMS
  const shellRoleLabel = isAdmin ? 'Admin' : 'Upload View'

  const [scope, setScope] = useState(COLUMN_SCOPES[0].scope)
  const active = COLUMN_SCOPES.find((s) => s.scope === scope) || COLUMN_SCOPES[0]

  return (
    <AppShell navItems={navItems} roleLabel={shellRoleLabel}>
      <div style={{ padding: 16, maxWidth: 1000 }}>
        <h2 style={{ fontWeight: 700, marginBottom: 4 }}>ตั้งค่า Format (รองรับการเปลี่ยนรูปแบบไฟล์)</h2>
        <p style={{ color: '#64748b', marginBottom: 20, fontSize: 14 }}>
          ตั้งค่าที่นี่มีผลทันทีตอนอัปโหลดครั้งถัดไป ไม่ต้องแก้โค้ดหรือรีสตาร์ทระบบ
        </p>

        <section style={{ marginBottom: 28 }}>
          <h3 style={{ fontWeight: 600, marginBottom: 8 }}>
            1) หัวคอลัมน์เปลี่ยน / เพิ่มใหม่ / สลับตำแหน่ง (Column Alias)
          </h3>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 12 }}>
            {COLUMN_SCOPES.map((s) => (
              <button
                key={s.scope}
                onClick={() => setScope(s.scope)}
                style={{
                  padding: '6px 12px',
                  borderRadius: 8,
                  border: '1px solid ' + (s.scope === scope ? '#2f5496' : '#cbd5e1'),
                  background: s.scope === scope ? '#2f5496' : '#fff',
                  color: s.scope === scope ? '#fff' : '#334155',
                  fontSize: 13,
                  cursor: 'pointer',
                }}
              >
                {s.label}
              </button>
            ))}
          </div>
          <ColumnAliasPanel scope={active.scope} targetHint={active.hint} />
        </section>

        <section>
          <h3 style={{ fontWeight: 600, marginBottom: 8 }}>
            2) รหัส P/N / S/N / Machine No. หน้างานเปลี่ยนรูปแบบ (Code Alias)
          </h3>
          <p style={{ color: '#64748b', marginBottom: 8, fontSize: 13 }}>
            เช่น หน้างานยิง <code>TNN-YN23993</code> หรือ <code>YN-2322#2</code> ให้ชี้กลับไปยัง S/N มาตรฐานในทะเบียนกลาง
          </p>
          <CodeAliasPanel componentType="it_controller" />
        </section>
      </div>
    </AppShell>
  )
}
