import { useMemo, useState } from 'react'
import AppShell from '../components/AppShell.jsx'
import SelectField from '../components/Selectfield.jsx'
import { ColumnAliasPanel, CodeAliasPanel } from '../components/FormatTools.jsx'
import { ADMIN_NAV_ITEMS } from './AdminDashboardpage.jsx'
import { FORMAT_NAV_ITEMS } from './MasterDataPage.jsx'
import { TagIcon, ArrowsRightLeftIcon } from '../components/icons.jsx'

// ─────────────────────────────────────────────────────────────────────────────
// หน้า "ตั้งค่า Format" — ให้หน้างาน/แอดมินรองรับการเปลี่ยน format ของไฟล์อัปโหลด
// และรหัส P/N / S/N / Machine No. ได้เองตอนรัน โดยไม่ต้องแก้โค้ด/รีดีพลอย
//
//  • Column Alias  = หัวคอลัมน์ในไฟล์เปลี่ยนชื่อ/เพิ่มใหม่/สลับตำแหน่ง → คอลัมน์มาตรฐาน
//    (ผูกกับ scope ของแต่ละ importer — backend รองรับทุก scope ด้านล่างแล้ว)
//  • Code Alias    = ค่า P/N / S/N / Machine No. รูปแบบใหม่หน้างาน → ค่ามาตรฐานในทะเบียน
//
// ปรับให้ใช้งานง่ายขึ้น (จากเดิมที่ต้องกดปุ่มเลือก 1 ใน 9 ปุ่มรวมกันแถวเดียว):
//   1) เลือก "หมวด" ก่อน (3 หมวดใหญ่ อ่านง่าย ไม่ต้องรู้ชื่อ importer ฝั่งระบบ)
//   2) แล้วค่อยเลือก "ไฟล์/งาน" ที่ต้องการตั้งค่าจาก dropdown ในหมวดนั้น
//   3) สีของหน้าใช้สี Theme เดียวกับทั้งระบบ (teal) แทนสีน้ำเงินเดิมที่ไม่ตรงธีม
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

// หมวดหมู่ — จัดกลุ่ม scope ให้เลือกง่ายเป็นขั้นตอน แทนปุ่ม 9 ปุ่มเรียงแถวเดียว
const CATEGORIES = [
  {
    id: 'upload',
    label: 'ไฟล์อัปโหลดประจำวัน',
    desc: 'ไฟล์ Planning / Warehouse / Engine ที่อัปโหลดเข้าเป็นประจำ',
    scopes: ['planning', 'wh1', 'wh2', 'engine'],
  },
  {
    id: 'master',
    label: 'ทะเบียนกลาง',
    desc: 'Master Data, Machine Spec, Serial List (IT Controller)',
    scopes: ['master_data', 'machine_spec', 'serial_list'],
  },
  {
    id: 'license',
    label: 'ใบอนุญาต',
    desc: 'ใบอนุญาตนำเข้า / ส่งออก',
    scopes: ['import_license', 'export_license'],
  },
]

export default function FormatSettingsPage() {
  const isAdmin = (localStorage.getItem('iconfirm_role') || '').toUpperCase() === 'ADMIN'
  const navItems = isAdmin ? ADMIN_NAV_ITEMS : FORMAT_NAV_ITEMS
  const shellRoleLabel = isAdmin ? 'Admin' : 'Upload View'

  const [categoryId, setCategoryId] = useState(CATEGORIES[0].id)
  const category = CATEGORIES.find((c) => c.id === categoryId) || CATEGORIES[0]

  const scopeOptions = useMemo(
    () =>
      category.scopes
        .map((s) => COLUMN_SCOPES.find((cs) => cs.scope === s))
        .filter(Boolean)
        .map((s) => ({ value: s.scope, label: s.label })),
    [category],
  )

  const [scope, setScope] = useState(scopeOptions[0]?.value)
  const active = COLUMN_SCOPES.find((s) => s.scope === scope) || COLUMN_SCOPES[0]

  function handleCategoryChange(id) {
    setCategoryId(id)
    const next = CATEGORIES.find((c) => c.id === id)
    const firstScope = next?.scopes?.[0]
    if (firstScope) setScope(firstScope)
  }

  return (
    <AppShell navItems={navItems} roleLabel={shellRoleLabel}>
      <div style={{ padding: 16, maxWidth: 960 }}>
        <h2 style={{ fontWeight: 700, marginBottom: 4, fontSize: 20, color: 'var(--color-brand-ink, #06312f)' }}>
          ตั้งค่า Format
        </h2>
        <p style={{ color: '#64748b', marginBottom: 20, fontSize: 14 }}>
          ตั้งค่าที่นี่มีผลทันทีตอนอัปโหลดครั้งถัดไป ไม่ต้องแก้โค้ดหรือรีสตาร์ทระบบ
        </p>

        {/* ขั้นที่ 1 — เลือกหมวด */}
        <div style={{ marginBottom: 14 }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#334155', marginBottom: 8 }}>
            1) เลือกหมวดที่ต้องการตั้งค่า
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {CATEGORIES.map((c) => {
              const isActive = c.id === categoryId
              return (
                <button
                  key={c.id}
                  onClick={() => handleCategoryChange(c.id)}
                  style={{
                    flex: '1 1 200px',
                    textAlign: 'left',
                    padding: '10px 14px',
                    borderRadius: 12,
                    border: '1.5px solid ' + (isActive ? 'var(--color-brand-500, #00cec8)' : '#e2e8f0'),
                    background: isActive ? 'var(--color-brand-50, #eafcfb)' : '#fff',
                    cursor: 'pointer',
                  }}
                >
                  <div style={{ fontSize: 14, fontWeight: 700, color: isActive ? 'var(--color-brand-700, #0f8580)' : '#0f172a' }}>
                    {c.label}
                  </div>
                  <div style={{ fontSize: 12, color: '#64748b', marginTop: 2 }}>{c.desc}</div>
                </button>
              )
            })}
          </div>
        </div>

        {/* ขั้นที่ 2 — เลือกไฟล์/งานในหมวดนั้น */}
        <div style={{ marginBottom: 22, maxWidth: 420 }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#334155', marginBottom: 8 }}>
            2) เลือกไฟล์/งานที่ต้องการตั้งค่า
          </div>
          <SelectField value={scope} onChange={setScope} options={scopeOptions} />
          {active?.hint && (
            <p style={{ fontSize: 12, color: '#94a3b8', marginTop: 6 }}>เช่นหัวคอลัมน์: {active.hint}</p>
          )}
        </div>

        <section style={{ marginBottom: 28 }}>
          <h3 style={{ fontWeight: 600, marginBottom: 8, display: 'flex', alignItems: 'center', gap: 6, fontSize: 15 }}>
            <TagIcon className="size-4" style={{ color: 'var(--color-brand-600, #00a39d)' }} />
            หัวคอลัมน์เปลี่ยน / เพิ่มใหม่ / สลับตำแหน่ง (Column Alias)
          </h3>
          <ColumnAliasPanel scope={active.scope} targetHint={active.hint} defaultOpen />
        </section>

        <section>
          <h3 style={{ fontWeight: 600, marginBottom: 8, display: 'flex', alignItems: 'center', gap: 6, fontSize: 15 }}>
            <ArrowsRightLeftIcon className="size-4" style={{ color: 'var(--color-brand-600, #00a39d)' }} />
            รหัส P/N / S/N / Machine No. หน้างานเปลี่ยนรูปแบบ (Code Alias)
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
