export default function QAPlaceholder({ title }) {
  return (
    <div className="qa-page-heading">
      <h2 className="wh-title">{title}</h2>
      <div className="dash-summary-card qa-placeholder-card">
        <p className="wh-subtitle">
          หน้านี้ยังไม่ได้ทำ — บอกรายละเอียดที่ต้องการได้เลย แล้วจะทำต่อให้ครับ
        </p>
      </div>
    </div>
  )
}