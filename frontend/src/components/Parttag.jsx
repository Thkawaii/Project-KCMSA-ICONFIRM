import './Parttag.css'

const PART_TAG_LABELS = {
  ITC: 'IT Controller',
  CV: 'Control Valve',
  SM: 'Swing Motor',
  MP: 'Motor Propel',
  PH: 'Pump Assy HYD',
  EN: 'Engine',
  CW: 'Counter Weight',
  MC: 'Machine',
}

export function partTagLabel(code) {
  const key = String(code || '').toUpperCase()
  return PART_TAG_LABELS[key] || ''
}

export default function PartTag({ code, label, className = '' }) {
  const key = String(code || '').toUpperCase()
  const text = label || partTagLabel(key)
  if (!text) return null

  const cls = ['part-tag', 'part-tag-' + (key ? key.toLowerCase() : 'mc'), className]
    .filter(Boolean)
    .join(' ')

  return <span className={cls}>{text}</span>
}
