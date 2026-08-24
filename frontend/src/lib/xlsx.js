// ─────────────────────────────────────────────────────────────────────────────
// xlsx.js — ตัวสร้างไฟล์ Excel (.xlsx) แบบ "ไม่มี dependency" พร้อมจัด Format ให้
//
// ทำไมไม่ใช้ไลบรารี (เช่น SheetJS):
//   • Docker build ใช้ `npm ci` ซึ่งผูกกับ package-lock — เพิ่ม dependency แล้ว
//     ถ้า lock ไม่ตรงจะ build พัง จึงเลี่ยงการเพิ่ม package ทั้งหมด
//   • .xlsx จริง ๆ คือไฟล์ ZIP ที่บรรจุ XML หลายไฟล์ — เขียนเองได้ในเบราว์เซอร์
//     โดยใช้ ZIP แบบ "store" (ไม่บีบอัด) + inline string (รองรับภาษาไทยผ่าน UTF-8)
//
// สิ่งที่ไฟล์นี้จัดให้อัตโนมัติ (ตาม theme สีของระบบ — teal #00a39d):
//   • Freeze Header (แถวแรกค้างด้านบนตอนเลื่อน)
//   • Header พื้นสี Theme + ตัวหนา + จัดกึ่งกลาง
//   • Filter ทุกคอลัมน์ (ผ่าน Excel Table ของจริง)
//   • Excel Table แถบสีสลับแถว (banded rows)
//   • ปรับความกว้าง Column อัตโนมัติตามความยาวเนื้อหา
//   • Border บาง ๆ รอบทุกเซลล์ในตาราง
//   • จัด Alignment ตามชนิดข้อมูล (number = กึ่งกลาง, text = ซ้าย, center = กึ่งกลาง)
//   • เซลล์ตัวเลขเก็บเป็น number จริง (ไม่ใช่ string) เพื่อให้ Format/SUM ใน Excel ใช้ได้
//   • รูปถ่าย (type:'image') ฝังลงในเซลล์ได้จริง (ผ่าน DrawingML) — ใช้กับ Check Sheet ของ QA
//
// วิธีใช้ (API — ชีตเดียว):
//   const blob = buildStyledXlsxBlob({
//     sheetName: 'QA Check Sheet',
//     columns: [
//       { key: 'item', header: 'ITEM', type: 'number', width: 8 },
//       { key: 'partName', header: 'Part Name', type: 'text' },
//       { key: 'photo', header: 'รูปถ่าย', type: 'image', width: 16 },
//       { key: 'status', header: 'Status', type: 'center' },
//     ],
//     rows: [{ item: 1, partName: 'ABC', photo: { bytes, ext: 'jpeg' }, status: 'Matched' }, ...],
//   })
//   downloadBlob(blob, 'qa-check-sheet.xlsx')
//
// วิธีใช้ (API — หลายชีต เช่น "แยกประเทศ"):
//   const blob = buildStyledXlsxWorkbookBlob({
//     sheets: [
//       { sheetName: 'JAPAN', columns, rows },
//       { sheetName: 'CHINA', columns, rows },
//     ],
//   })
//
// ค่าเซลล์ชนิด image รับได้ 2 แบบ:
//   • { bytes: Uint8Array, ext: 'jpeg'|'png', wpx?, hpx? }
//   • { dataUrl: 'data:image/jpeg;base64,...', wpx?, hpx? }
//   (ถ้าไม่มีรูป ให้ใส่ค่า null/'' ได้ — เซลล์จะว่างไว้เฉย ๆ)
// ─────────────────────────────────────────────────────────────────────────────

// ── CRC-32 (สำหรับ ZIP) ─────────────────────────────────────────────────────
const CRC_TABLE = (() => {
  const table = new Uint32Array(256)
  for (let n = 0; n < 256; n++) {
    let c = n
    for (let k = 0; k < 8; k++) {
      c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
    }
    table[n] = c >>> 0
  }
  return table
})()

function crc32(bytes) {
  let crc = 0xffffffff
  for (let i = 0; i < bytes.length; i++) {
    crc = (crc >>> 8) ^ CRC_TABLE[(crc ^ bytes[i]) & 0xff]
  }
  return (crc ^ 0xffffffff) >>> 0
}

const utf8 = (str) => new TextEncoder().encode(str)

// escape อักขระพิเศษของ XML
function xmlEscape(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    // ตัดอักขระควบคุมที่ XML 1.0 ไม่รับ (กันไฟล์เสีย)
    .replace(/[\x00-\x08\x0B\x0C\x0E-\x1F]/g, '')
}

// เลขคอลัมน์ (1-based) -> ตัวอักษร (1->A, 27->AA)
function colLetter(n) {
  let s = ''
  while (n > 0) {
    const m = (n - 1) % 26
    s = String.fromCharCode(65 + m) + s
    n = Math.floor((n - 1) / 26)
  }
  return s
}

// px -> EMU (English Metric Unit) — DrawingML วัดขนาด/ตำแหน่งเป็น EMU (1px = 9525 EMU)
const EMU_PER_PX = 9525

// ── ZIP แบบ store (ไม่บีบอัด) จากรายการไฟล์ ─────────────────────────────────
// รองรับทั้งไฟล์ข้อความ (f.content = string) และไฟล์ไบนารี (f.bytes = Uint8Array)
// เช่น รูปถ่ายใน xl/media/*
function makeZip(files) {
  const encoded = files.map((f) => {
    const nameBytes = utf8(f.name)
    const dataBytes = f.bytes ? f.bytes : utf8(f.content || '')
    return { nameBytes, dataBytes, crc: crc32(dataBytes) }
  })

  const chunks = []
  const localOffsets = []
  let offset = 0

  const u16 = (n) => {
    const b = new Uint8Array(2)
    new DataView(b.buffer).setUint16(0, n, true)
    return b
  }
  const u32 = (n) => {
    const b = new Uint8Array(4)
    new DataView(b.buffer).setUint32(0, n >>> 0, true)
    return b
  }
  const push = (bytes) => {
    chunks.push(bytes)
    offset += bytes.length
  }

  encoded.forEach((f) => {
    localOffsets.push(offset)
    push(u32(0x04034b50))
    push(u16(20))
    push(u16(0))
    push(u16(0))
    push(u16(0))
    push(u16(0))
    push(u32(f.crc))
    push(u32(f.dataBytes.length))
    push(u32(f.dataBytes.length))
    push(u16(f.nameBytes.length))
    push(u16(0))
    push(f.nameBytes)
    push(f.dataBytes)
  })

  const cdStart = offset
  encoded.forEach((f, i) => {
    push(u32(0x02014b50))
    push(u16(20))
    push(u16(20))
    push(u16(0))
    push(u16(0))
    push(u16(0))
    push(u16(0))
    push(u32(f.crc))
    push(u32(f.dataBytes.length))
    push(u32(f.dataBytes.length))
    push(u16(f.nameBytes.length))
    push(u16(0))
    push(u16(0))
    push(u16(0))
    push(u16(0))
    push(u32(0))
    push(u32(localOffsets[i]))
    push(f.nameBytes)
  })
  const cdSize = offset - cdStart

  push(u32(0x06054b50))
  push(u16(0))
  push(u16(0))
  push(u16(encoded.length))
  push(u16(encoded.length))
  push(u32(cdSize))
  push(u32(cdStart))
  push(u16(0))

  return new Blob(chunks, {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  })
}

// ── สไตล์ (styles.xml) ──────────────────────────────────────────────────────
// อิงสี Theme ของระบบ (ดู theme.css): --color-brand-600 #00a39d (header),
// --color-brand-50 #eafcfb (แถบแถวสลับ)
const HEADER_FILL_RGB = 'FF00A39D'
const BAND_FILL_RGB = 'FFEAFCFB'
const BORDER_RGB = 'FFD7E1E8'
const FONT_NAME = 'Tahoma' // รองรับภาษาไทยและมีติดตั้งอยู่แทบทุกเครื่อง Windows

// index ของ cellXfs ที่ใช้อ้างจาก worksheet (s="N")
const XF = {
  BASE: 0,
  HEADER: 1,
  TEXT: 2,
  TEXT_BAND: 3,
  CENTER: 4,
  CENTER_BAND: 5,
  NUMBER: 6,
  NUMBER_BAND: 7,
}

function buildStylesXml() {
  return (
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
    '<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">' +
    '<fonts count="2">' +
    `<font><sz val="11"/><name val="${FONT_NAME}"/></font>` +
    `<font><b/><sz val="11"/><color rgb="FFFFFFFF"/><name val="${FONT_NAME}"/></font>` +
    '</fonts>' +
    '<fills count="4">' +
    '<fill><patternFill patternType="none"/></fill>' +
    '<fill><patternFill patternType="gray125"/></fill>' +
    `<fill><patternFill patternType="solid"><fgColor rgb="${HEADER_FILL_RGB}"/><bgColor indexed="64"/></patternFill></fill>` +
    `<fill><patternFill patternType="solid"><fgColor rgb="${BAND_FILL_RGB}"/><bgColor indexed="64"/></patternFill></fill>` +
    '</fills>' +
    '<borders count="2">' +
    '<border><left/><right/><top/><bottom/><diagonal/></border>' +
    `<border><left style="thin"><color rgb="${BORDER_RGB}"/></left><right style="thin"><color rgb="${BORDER_RGB}"/></right><top style="thin"><color rgb="${BORDER_RGB}"/></top><bottom style="thin"><color rgb="${BORDER_RGB}"/></bottom><diagonal/></border>` +
    '</borders>' +
    '<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>' +
    '<cellXfs count="8">' +
    '<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>' +
    '<xf numFmtId="0" fontId="1" fillId="2" borderId="1" xfId="0" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf>' +
    '<xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyBorder="1" applyAlignment="1"><alignment horizontal="left" vertical="center" wrapText="1"/></xf>' +
    '<xf numFmtId="0" fontId="0" fillId="3" borderId="1" xfId="0" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="left" vertical="center" wrapText="1"/></xf>' +
    '<xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf>' +
    '<xf numFmtId="0" fontId="0" fillId="3" borderId="1" xfId="0" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf>' +
    '<xf numFmtId="1" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center"/></xf>' +
    '<xf numFmtId="1" fontId="0" fillId="3" borderId="1" xfId="0" applyNumberFormat="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center"/></xf>' +
    '</cellXfs>' +
    '<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>' +
    '</styleSheet>'
  )
}

// เลือก style index ของเซลล์ข้อมูล ตามชนิดคอลัมน์ + ว่าเป็นแถวแถบสีหรือไม่
function dataXf(type, banded) {
  if (type === 'number') return banded ? XF.NUMBER_BAND : XF.NUMBER
  if (type === 'center' || type === 'image') return banded ? XF.CENTER_BAND : XF.CENTER
  return banded ? XF.TEXT_BAND : XF.TEXT
}

// ประมาณความกว้างคอลัมน์จากความยาวเนื้อหา (ตัวอักษรไทยกว้างกว่าอังกฤษ ~1.6 เท่า)
function estimateWidth(text) {
  const s = String(text ?? '')
  let w = 0
  for (const ch of s) {
    w += /[\u0E00-\u0E7F]/.test(ch) ? 1.6 : 1
  }
  return w
}

// ขนาดรูปในเซลล์ (px) — ค่าเริ่มต้นถ้าไม่ได้ระบุมากับข้อมูล
const IMAGE_CELL_PX = 88
// ความกว้างคอลัมน์รูป (หน่วย Excel char) ให้พอดีกับรูป ~88px + ระยะขอบ
const IMAGE_COL_WIDTH = 14
// ความสูงแถวที่มีรูป (points) — 88px ≈ 66pt
const IMAGE_ROW_HEIGHT_PT = 70

// อ่านค่าเซลล์รูป -> { bytes, ext } (คืน null ถ้าไม่มี/ไม่รองรับ)
function normalizeImage(val) {
  if (!val) return null
  if (val.bytes && val.bytes.length) {
    const ext = (val.ext || 'jpeg').toLowerCase() === 'png' ? 'png' : 'jpeg'
    return { bytes: val.bytes, ext, wpx: val.wpx, hpx: val.hpx }
  }
  if (val.dataUrl) {
    const parsed = dataUrlToBytes(val.dataUrl)
    if (!parsed) return null
    return { bytes: parsed.bytes, ext: parsed.ext, wpx: val.wpx, hpx: val.hpx }
  }
  return null
}

// data:image/xxx;base64,.... -> { bytes, ext } (รองรับเฉพาะ jpeg/png)
export function dataUrlToBytes(dataUrl) {
  const m = /^data:image\/([a-zA-Z0-9+.-]+);base64,(.*)$/.exec(dataUrl || '')
  if (!m) return null
  let ext = m[1].toLowerCase()
  if (ext === 'jpg') ext = 'jpeg'
  if (ext !== 'jpeg' && ext !== 'png') return null // ชนิดอื่นฝังใน xlsx ไม่ได้ตรง ๆ
  const bin = atob(m[2])
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return { bytes, ext }
}

function buildSheetXml({ columns, rows, freezeHeader, hasImages, imageRowPx }) {
  const colCount = columns.length
  const rowCount = rows.length

  let colsXml = '<cols>'
  columns.forEach((col, i) => {
    let width = col.width
    if (!width) {
      if (col.type === 'image') {
        width = IMAGE_COL_WIDTH
      } else {
        let maxLen = estimateWidth(col.header)
        rows.forEach((r) => {
          maxLen = Math.max(maxLen, estimateWidth(r[col.key]))
        })
        width = Math.min(Math.max(maxLen + 3, 10), 42)
      }
    }
    colsXml += `<col min="${i + 1}" max="${i + 1}" width="${width.toFixed(2)}" customWidth="1"/>`
  })
  colsXml += '</cols>'

  const sheetViewsXml = freezeHeader
    ? '<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/>' +
      '<selection pane="bottomLeft" activeCell="A2" sqref="A2"/></sheetView></sheetViews>'
    : '<sheetViews><sheetView workbookViewId="0"/></sheetViews>'

  let headerCells = ''
  columns.forEach((col, c) => {
    const ref = colLetter(c + 1) + '1'
    headerCells += `<c r="${ref}" t="inlineStr" s="${XF.HEADER}"><is><t xml:space="preserve">${xmlEscape(col.header)}</t></is></c>`
  })
  let body = `<row r="1" ht="22" customHeight="1">${headerCells}</row>`

  const rowHeightPx = imageRowPx || IMAGE_CELL_PX
  const rowHtAttr = hasImages ? ` ht="${(rowHeightPx * 0.75).toFixed(2)}" customHeight="1"` : ''

  rows.forEach((row, r) => {
    const rowNum = r + 2
    const banded = r % 2 === 1
    let cells = ''
    columns.forEach((col, c) => {
      const ref = colLetter(c + 1) + rowNum
      const val = row[col.key]
      const s = dataXf(col.type, banded)
      if (col.type === 'image') {
        // เซลล์รูปเว้นค่าไว้ (รูปวาดผ่าน DrawingML) — คงพื้น/ขอบของเซลล์ไว้
        cells += `<c r="${ref}" s="${s}"/>`
      } else if (val == null || val === '') {
        cells += `<c r="${ref}" s="${s}"/>`
      } else if (col.type === 'number' && typeof val === 'number' && Number.isFinite(val)) {
        cells += `<c r="${ref}" s="${s}"><v>${val}</v></c>`
      } else {
        cells += `<c r="${ref}" t="inlineStr" s="${s}"><is><t xml:space="preserve">${xmlEscape(val)}</t></is></c>`
      }
    })
    body += `<row r="${rowNum}"${rowHtAttr}>${cells}</row>`
  })

  const dim = `A1:${colLetter(colCount)}${rowCount + 1}`

  // ลำดับ element ใน worksheet ต้องถูกตาม schema: ... sheetData ... drawing ... tableParts
  return (
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
    '<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">' +
    `<dimension ref="${dim}"/>` +
    sheetViewsXml +
    colsXml +
    `<sheetData>${body}</sheetData>` +
    (hasImages ? '<drawing r:id="rIdDrawing"/>' : '') +
    (rowCount > 0 ? '<tableParts count="1"><tablePart r:id="rIdTable"/></tableParts>' : '') +
    '</worksheet>'
  )
}

// ── Excel Table (ListObject) — ให้ Filter ทุกคอลัมน์ + แถบสีสลับแถวจริงจาก Excel ─
function buildTableXml({ columns, tableRef, tableId, tableName }) {
  let tableColumns = '<tableColumns count="' + columns.length + '">'
  columns.forEach((col, i) => {
    tableColumns += `<tableColumn id="${i + 1}" name="${xmlEscape(col.header)}"/>`
  })
  tableColumns += '</tableColumns>'

  return (
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
    '<table xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ' +
    `id="${tableId}" name="${xmlEscape(tableName)}" displayName="${xmlEscape(tableName)}" ref="${tableRef}" totalsRowShown="0">` +
    `<autoFilter ref="${tableRef}"/>` +
    tableColumns +
    // showRowStripes="0" เพราะแถบสีสลับแถวคุมเองผ่าน cellXfs แล้ว (สีตรง Theme แน่นอน
    // ทุกโปรแกรม ไม่ขึ้นกับชุดสีของ TableStyle ที่ Excel เลือกให้)
    '<tableStyleInfo name="TableStyleLight1" showFirstColumn="0" showLastColumn="0" showRowStripes="0" showColumnStripes="0"/>' +
    '</table>'
  )
}

// ── DrawingML — วางรูปแต่ละรูปลงในเซลล์ (oneCellAnchor, ขนาดคงที่ + ระยะขอบ) ──
// picList: [{ col, row, blipRel, wpx, hpx }]  (col,row = 0-based ของเซลล์เป้าหมาย)
function buildDrawingXml(picList) {
  const NS =
    'xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" ' +
    'xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ' +
    'xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"'

  let anchors = ''
  picList.forEach((p, i) => {
    const wpx = p.wpx || IMAGE_CELL_PX
    const hpx = p.hpx || IMAGE_CELL_PX
    const padPx = 3
    const cx = Math.round(wpx * EMU_PER_PX)
    const cy = Math.round(hpx * EMU_PER_PX)
    const off = Math.round(padPx * EMU_PER_PX)
    const picId = i + 2 // เริ่มที่ 2 กันชนกับ id ที่บางโปรแกรมสงวน
    anchors +=
      '<xdr:oneCellAnchor>' +
      `<xdr:from><xdr:col>${p.col}</xdr:col><xdr:colOff>${off}</xdr:colOff>` +
      `<xdr:row>${p.row}</xdr:row><xdr:rowOff>${off}</xdr:rowOff></xdr:from>` +
      `<xdr:ext cx="${cx}" cy="${cy}"/>` +
      '<xdr:pic>' +
      '<xdr:nvPicPr>' +
      `<xdr:cNvPr id="${picId}" name="Picture ${picId}"/>` +
      '<xdr:cNvPicPr><a:picLocks noChangeAspect="1"/></xdr:cNvPicPr>' +
      '</xdr:nvPicPr>' +
      `<xdr:blipFill><a:blip r:embed="${p.blipRel}"/><a:stretch><a:fillRect/></a:stretch></xdr:blipFill>` +
      '<xdr:spPr>' +
      '<a:xfrm><a:off x="0" y="0"/><a:ext cx="' + cx + '" cy="' + cy + '"/></a:xfrm>' +
      '<a:prstGeom prst="rect"><a:avLst/></a:prstGeom>' +
      '</xdr:spPr>' +
      '</xdr:pic>' +
      '<xdr:clientData/>' +
      '</xdr:oneCellAnchor>'
  })

  return (
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
    `<xdr:wsDr ${NS}>` +
    anchors +
    '</xdr:wsDr>'
  )
}

// ── ประกอบ 1 workbook (หลายชีต) เป็นรายการไฟล์ .xlsx ──────────────────────────
function buildWorkbookParts(sheets) {
  // สะสมสื่อ (รูป) ระดับ workbook เพื่อให้ชื่อไฟล์ไม่ชนกันข้ามชีต
  const media = [] // { name, bytes }  (name = imageN.ext)
  let mediaSeq = 0
  const usedExts = new Set() // สำหรับประกาศ Default Extension ใน content types

  // เตรียมข้อมูลต่อชีต
  const prepared = sheets.map((raw, idx) => {
    const sheetIdx = idx + 1
    const safeName = (raw.sheetName || `Sheet${sheetIdx}`)
      .slice(0, 31)
      .replace(/[\\/?*[\]:]/g, ' ')
    const columns = (raw.columns || []).map((c) => ({
      key: c.key,
      header: c.header ?? c.key,
      type: c.type || 'text',
      width: c.width,
    }))
    const rows = raw.rows || []
    const colCount = columns.length
    const rowCount = rows.length

    // รวบรวมรูปจากคอลัมน์ชนิด image
    const imageCols = columns
      .map((c, ci) => ({ c, ci }))
      .filter((x) => x.c.type === 'image')

    const picList = []
    let maxImgPx = 0
    if (imageCols.length) {
      rows.forEach((row, ri) => {
        imageCols.forEach(({ c, ci }) => {
          const img = normalizeImage(row[c.key])
          if (!img) return
          mediaSeq += 1
          const name = `image${mediaSeq}.${img.ext}`
          media.push({ name, bytes: img.bytes })
          usedExts.add(img.ext)
          const wpx = img.wpx || IMAGE_CELL_PX
          const hpx = img.hpx || IMAGE_CELL_PX
          maxImgPx = Math.max(maxImgPx, hpx)
          picList.push({
            col: ci,
            row: ri + 1, // +1 ข้ามแถว header
            mediaName: name,
            wpx,
            hpx,
          })
        })
      })
    }
    const hasImages = picList.length > 0

    return {
      sheetIdx,
      safeName,
      columns,
      rows,
      colCount,
      rowCount,
      picList,
      hasImages,
      imageRowPx: maxImgPx || IMAGE_CELL_PX,
    }
  })

  const parts = []

  // workbook.xml — ลิสต์ชีตทั้งหมด
  let sheetsXml = '<sheets>'
  prepared.forEach((s) => {
    sheetsXml += `<sheet name="${xmlEscape(s.safeName)}" sheetId="${s.sheetIdx}" r:id="rIdSheet${s.sheetIdx}"/>`
  })
  sheetsXml += '</sheets>'
  parts.push({
    name: 'xl/workbook.xml',
    content:
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
      '<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">' +
      sheetsXml +
      '</workbook>',
  })

  // workbook rels — worksheet ทุกตัว + styles
  let wbRels = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
    '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
  prepared.forEach((s) => {
    wbRels += `<Relationship Id="rIdSheet${s.sheetIdx}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet${s.sheetIdx}.xml"/>`
  })
  wbRels += '<Relationship Id="rIdStyles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>'
  wbRels += '</Relationships>'
  parts.push({ name: 'xl/_rels/workbook.xml.rels', content: wbRels })

  parts.push({ name: 'xl/styles.xml', content: buildStylesXml() })

  // ต่อชีต: worksheet + rels + table + drawing
  const contentOverrides = []
  prepared.forEach((s) => {
    parts.push({
      name: `xl/worksheets/sheet${s.sheetIdx}.xml`,
      content: buildSheetXml({
        columns: s.columns,
        rows: s.rows,
        freezeHeader: true,
        hasImages: s.hasImages,
        imageRowPx: s.imageRowPx,
      }),
    })
    contentOverrides.push(
      `<Override PartName="/xl/worksheets/sheet${s.sheetIdx}.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`,
    )

    // rels ของ worksheet (table + drawing ถ้ามี)
    const needSheetRels = s.rowCount > 0 || s.hasImages
    if (needSheetRels) {
      let rels = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
      if (s.rowCount > 0) {
        rels += `<Relationship Id="rIdTable" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/table" Target="../tables/table${s.sheetIdx}.xml"/>`
      }
      if (s.hasImages) {
        rels += `<Relationship Id="rIdDrawing" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/drawing${s.sheetIdx}.xml"/>`
      }
      rels += '</Relationships>'
      parts.push({ name: `xl/worksheets/_rels/sheet${s.sheetIdx}.xml.rels`, content: rels })
    }

    if (s.rowCount > 0) {
      const tableRef = `A1:${colLetter(s.colCount)}${s.rowCount + 1}`
      parts.push({
        name: `xl/tables/table${s.sheetIdx}.xml`,
        content: buildTableXml({
          columns: s.columns,
          tableRef,
          tableId: s.sheetIdx,
          tableName: `Table${s.sheetIdx}`,
        }),
      })
      contentOverrides.push(
        `<Override PartName="/xl/tables/table${s.sheetIdx}.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.table+xml"/>`,
      )
    }

    if (s.hasImages) {
      // drawing rels: blip แต่ละรูป -> media
      let drawRels = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
      const picWithRel = s.picList.map((p, i) => {
        const blipRel = `rIdImg${i + 1}`
        drawRels += `<Relationship Id="${blipRel}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/${p.mediaName}"/>`
        return { ...p, blipRel }
      })
      drawRels += '</Relationships>'

      parts.push({
        name: `xl/drawings/drawing${s.sheetIdx}.xml`,
        content: buildDrawingXml(picWithRel),
      })
      parts.push({
        name: `xl/drawings/_rels/drawing${s.sheetIdx}.xml.rels`,
        content: drawRels,
      })
      contentOverrides.push(
        `<Override PartName="/xl/drawings/drawing${s.sheetIdx}.xml" ContentType="application/vnd.openxmlformats-officedocument.drawing+xml"/>`,
      )
    }
  })

  // media (รูป) — ไฟล์ไบนารี
  media.forEach((m) => {
    parts.push({ name: `xl/media/${m.name}`, bytes: m.bytes })
  })

  // [Content_Types].xml
  let defaults =
    '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>' +
    '<Default Extension="xml" ContentType="application/xml"/>'
  if (usedExts.has('png')) defaults += '<Default Extension="png" ContentType="image/png"/>'
  if (usedExts.has('jpeg')) defaults += '<Default Extension="jpeg" ContentType="image/jpeg"/>'
  const contentTypes =
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
    '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">' +
    defaults +
    '<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>' +
    '<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>' +
    contentOverrides.join('') +
    '</Types>'
  parts.push({ name: '[Content_Types].xml', content: contentTypes })

  // _rels/.rels
  parts.push({
    name: '_rels/.rels',
    content:
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
      '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">' +
      '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>' +
      '</Relationships>',
  })

  return parts
}

// ── API หลัก (ชีตเดียว) ─────────────────────────────────────────────────────
// columns: [{ key, header, type: 'text' | 'number' | 'center' | 'image', width? }]
// rows:    [{ [key]: value }, ...]
export function buildStyledXlsxBlob({ sheetName, columns, rows }) {
  return buildStyledXlsxWorkbookBlob({
    sheets: [{ sheetName, columns, rows: rows || [] }],
  })
}

// ── API หลัก (หลายชีต) ──────────────────────────────────────────────────────
// sheets: [{ sheetName, columns, rows }]
export function buildStyledXlsxWorkbookBlob({ sheets }) {
  const list = (sheets && sheets.length ? sheets : [{ sheetName: 'Sheet1', columns: [], rows: [] }]).map(
    (s) => ({
      sheetName: s.sheetName,
      columns: s.columns || [],
      rows: s.rows || [],
    }),
  )
  return makeZip(buildWorkbookParts(list))
}

// ── รูปแบบเดิม (ยังคงไว้เพื่อความเข้ากันได้) ─────────────────────────────────
// sheetToXlsxBlob(sheetName, [header, ...bodyRows]) — แปลง rows แบบ array เป็น
// columns/rows แบบ object ให้อัตโนมัติ แล้วส่งผ่าน builder ใหม่ (จึงได้ format ครบเช่นกัน)
export function sheetToXlsxBlob(sheetName, rows) {
  const [header, ...body] = rows
  const columns = (header || []).map((h, i) => ({
    key: `c${i}`,
    header: h,
    type: body.some((r) => typeof r[i] === 'number') ? 'number' : 'text',
  }))
  const objRows = body.map((r) => {
    const o = {}
    columns.forEach((col, i) => {
      o[col.key] = r[i]
    })
    return o
  })
  return buildStyledXlsxBlob({ sheetName, columns, rows: objRows })
}

// สั่งดาวน์โหลด Blob เป็นไฟล์ (รองรับมือถือ: เปิดแท็บใหม่ถ้าดาวน์โหลดตรงไม่ได้)
export function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 60_000)
}
