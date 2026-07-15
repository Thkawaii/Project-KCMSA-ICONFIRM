import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getMasterData, getTsfScans, createTsfScan } from '../api/tsf.js'
import { logout } from '../api/auth.js'

export default function TSFOperatorPage() {
  const navigate = useNavigate()
  const [masterData, setMasterData] = useState([])
  const [serialNumber, setSerialNumber] = useState('')
  const [partNo, setPartNo] = useState('')
  const [specCode, setSpecCode] = useState('')
  const [photo, setPhoto] = useState(null)
  const [photoPreview, setPhotoPreview] = useState('')
  const [submissions, setSubmissions] = useState([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [formError, setFormError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function loadData() {
    setLoading(true)
    setLoadError('')
    try {
      const [master, scans] = await Promise.all([getMasterData(), getTsfScans()])
      setMasterData(master || [])
      setSubmissions(scans || [])
    } catch (err) {
      setLoadError(err.message || 'โหลดข้อมูลไม่สำเร็จ')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  function handlePhotoChange(e) {
    const file = e.target.files?.[0]
    if (!file) return
    setPhoto(file)
    setPhotoPreview(URL.createObjectURL(file))
  }

  function resetForm() {
    setSerialNumber('')
    setPartNo('')
    setSpecCode('')
    setPhoto(null)
    setPhotoPreview('')
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setFormError('')

    if (!serialNumber || !partNo || !specCode) {
      setFormError('กรุณากรอก Serial Number, Part No. และ Spec Code ให้ครบ')
      return
    }
    if (!photo) {
      setFormError('กรุณาแนบรูปถ่ายยืนยันการติดตั้ง')
      return
    }

    const master = masterData.find((m) => m.PartNo === partNo)
    // เทียบเบื้องต้นฝั่ง frontend ว่ามี part_no นี้อยู่ใน master data ไหม
    // ผล PASS/FAIL จริงจะถูกยืนยันอีกทีโดย QA (ผ่าน /qa-confirm)
    const matchesMaster = Boolean(master) && master.SpecCode === specCode

    // หมายเหตุ: backend เก็บแค่ชื่อไฟล์ (FileName) ไม่มี endpoint อัปโหลดรูปจริง
    // ถ้าต้องเก็บไฟล์รูปจริงต้องเพิ่ม multipart upload endpoint ฝั่ง backend ก่อน
    const payload = {
      SerialNumber: serialNumber,
      ActualPartNo: partNo,
      ActualSpecCode: specCode,
      ValidationStatus: matchesMaster ? 'Matched' : 'Mismatch',
      FileName: photo.name,
      ScannedBy: localStorage.getItem('iconfirm_name') || '',
    }

    setSubmitting(true)
    try {
      await createTsfScan(payload)
      resetForm()
      await loadData()
    } catch (err) {
      setFormError(err.message || 'บันทึกไม่สำเร็จ')
    } finally {
      setSubmitting(false)
    }
  }

  function handleLogout() {
    logout()
    navigate('/login')
  }

  return (
    <div className="wh-page">
      <header className="wh-topbar">
        <div className="brand-row">
          <span className="brand-badge">KOBELCO</span>
          <h1 className="brand-title-sm">I-CONFIRM</h1>
        </div>
        <button className="wh-logout-btn" onClick={handleLogout}>
          Logout
        </button>
      </header>

      <main className="wh-main">
        <div className="wh-heading-row">
          <div>
            <h2 className="wh-title">TSF Operator — สแกนและบันทึกข้อมูลชิ้นส่วน</h2>
            <p className="wh-subtitle">
              สแกน Part No. / Serial No. และแนบรูปถ่ายหลังติดตั้ง เพื่อส่งให้ QA ตรวจสอบยืนยัน
            </p>
          </div>
        </div>

        {loadError && (
          <p className="form-error" role="alert">
            {loadError}
          </p>
        )}

        <div className="tsf-layout">
          <form className="tsf-form-card" onSubmit={handleSubmit}>
            <h3 className="tsf-form-title">สแกน / บันทึกข้อมูล</h3>

            <label className="wh-modal-label" htmlFor="serialNumber">
              Serial Number (S/N)
            </label>
            <input
              id="serialNumber"
              className="wh-modal-input"
              type="text"
              placeholder="สแกนหรือกรอก S/N"
              value={serialNumber}
              onChange={(e) => setSerialNumber(e.target.value)}
            />

            <label className="wh-modal-label" htmlFor="partNo">
              Part Number (P/N)
            </label>
            <input
              id="partNo"
              className="wh-modal-input"
              type="text"
              placeholder="เช่น CV001"
              value={partNo}
              onChange={(e) => setPartNo(e.target.value)}
              list="master-part-no"
            />
            <datalist id="master-part-no">
              {masterData.map((m) => (
                <option key={m.PartNo} value={m.PartNo} />
              ))}
            </datalist>

            <label className="wh-modal-label" htmlFor="specCode">
              Specification Code
            </label>
            <input
              id="specCode"
              className="wh-modal-input"
              type="text"
              placeholder="เช่น SPEC001"
              value={specCode}
              onChange={(e) => setSpecCode(e.target.value)}
            />

            <label className="wh-modal-label" htmlFor="photo">
              รูปถ่ายยืนยันการติดตั้ง
            </label>
            <input id="photo" type="file" accept="image/*" onChange={handlePhotoChange} />
            {photoPreview && (
              <img className="tsf-photo-preview" src={photoPreview} alt="preview" />
            )}

            {formError && (
              <p className="form-error" role="alert">
                {formError}
              </p>
            )}

            <button type="submit" className="wh-issue-btn tsf-submit-btn" disabled={submitting}>
              {submitting ? 'กำลังบันทึก...' : 'บันทึกและส่งให้ QA'}
            </button>
          </form>

          <div className="tsf-table-card">
            <h3 className="tsf-form-title">รายการที่ส่งแล้ว ({submissions.length})</h3>
            <table className="wh-table">
              <thead>
                <tr>
                  <th>S/N</th>
                  <th>P/N</th>
                  <th>Spec Code</th>
                  <th>รูปถ่าย</th>
                  <th>สถานะ</th>
                </tr>
              </thead>
              <tbody>
                {loading && (
                  <tr>
                    <td colSpan={5} className="wh-empty-cell">
                      กำลังโหลดข้อมูล...
                    </td>
                  </tr>
                )}
                {!loading &&
                  submissions.map((s) => (
                    <tr key={s.ID}>
                      <td>{s.SerialNumber}</td>
                      <td>{s.ActualPartNo}</td>
                      <td>{s.ActualSpecCode}</td>
                      <td>{s.FileName}</td>
                      <td>
                        <span
                          className={
                            s.ValidationStatus === 'Matched' ? 'wh-qty-badge' : 'wh-qty-badge wh-qty-zero'
                          }
                        >
                          {s.ValidationStatus}
                        </span>
                      </td>
                    </tr>
                  ))}
                {!loading && submissions.length === 0 && (
                  <tr>
                    <td colSpan={5} className="wh-empty-cell">
                      ยังไม่มีรายการที่ส่ง
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </main>
    </div>
  )
}