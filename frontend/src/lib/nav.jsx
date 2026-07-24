import { createContext, useCallback, useContext, useState } from 'react'

// เดิมระบบใช้ react-router (BrowserRouter) ทำให้ path จริงขึ้น address bar
// เช่น /warehouse, /qa/machine/ABC — ตอนนี้เปลี่ยนมาใช้ context ตัวนี้แทน
// สลับหน้าด้วย React state ล้วน ๆ ไม่มีการ pushState/เปลี่ยน URL เลย
// ผลคือ address bar จะค้างอยู่ path เดียว (เช่น http://localhost:9004/) ตลอดทั้งระบบ
// และปุ่ม back/forward ของ browser จะไม่มี "หน้าเดิมของแอป" ให้ย้อนกลับไปเห็นได้อีก
const NavContext = createContext(null)

export function NavProvider({ initialView, children }) {
  const [view, setView] = useState(initialView)
  const [params, setParams] = useState({})

  const navigate = useCallback((nextView, nextParams = {}) => {
    setView(nextView)
    setParams(nextParams)
  }, [])

  return (
    <NavContext.Provider value={{ view, params, navigate }}>
      {children}
    </NavContext.Provider>
  )
}

function useNavContext() {
  const ctx = useContext(NavContext)
  if (!ctx) throw new Error('useNav hooks ต้องถูกเรียกภายใน <NavProvider> เท่านั้น')
  return ctx
}

// drop-in แทน useNavigate() ของ react-router — เรียกแบบเดิมได้เลย navigate('/warehouse')
export function useAppNavigate() {
  return useNavContext().navigate
}

// drop-in แทน useParams() ของ react-router
export function useAppParams() {
  return useNavContext().params
}

// ใช้เช็คว่าตอนนี้อยู่ view ไหน (เช่น ไฮไลต์เมนูที่ active)
export function useAppView() {
  return useNavContext().view
}
