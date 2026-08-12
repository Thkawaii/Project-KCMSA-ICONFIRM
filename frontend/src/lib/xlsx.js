// ─────────────────────────────────────────────────────────────────────────────
// xlsx.js — ตัวสร้างไฟล์ Excel (.xlsx) แบบ "ไม่มี dependency"
//
// ทำไมไม่ใช้ไลบรารี (เช่น SheetJS):
//   • Docker build ใช้ `npm ci` ซึ่งผูกกับ package-lock — เพิ่ม dependency แล้ว
//     ถ้า lock ไม่ตรงจะ build พัง จึงเลี่ยงการเพิ่ม package ทั้งหมด
//   • .xlsx จริง ๆ คือไฟล์ ZIP ที่บรรจุ XML หลายไฟล์ — เขียนเองได้ในเบราว์เซอร์
//     โดยใช้ ZIP แบบ "store" (ไม่บีบอัด) + inline string (รองรับภาษาไทยผ่าน UTF-8)
//
// วิธีใช้:
//   const blob = sheetToXlsxBlob('QA', [
//     ['ITEM', 'Part Name', ...],   // แถวหัว
//     [1, 'ชื่อพาร์ท', ...],        // แถวข้อมูล (string หรือ number)
//   ])
//   downloadBlob(blob, 'qa-check-sheet.xlsx')
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

// สร้าง XML ของ worksheet จาก rows (array ของ array; cell = string | number | null)
function buildSheetXml(rows) {
  let body = ''
  rows.forEach((row, r) => {
    const rowNum = r + 1
    let cells = ''
    row.forEach((val, c) => {
      const ref = colLetter(c + 1) + rowNum
      if (val == null || val === '') return // เว้นเซลล์ว่างไว้ (ไม่ต้องเขียน)
      if (typeof val === 'number' && Number.isFinite(val)) {
        cells += `<c r="${ref}"><v>${val}</v></c>`
      } else {
        cells += `<c r="${ref}" t="inlineStr"><is><t xml:space="preserve">${xmlEscape(val)}</t></is></c>`
      }
    })
    body += `<row r="${rowNum}">${cells}</row>`
  })

  return (
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
    '<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">' +
    `<sheetData>${body}</sheetData>` +
    '</worksheet>'
  )
}

function buildParts(sheetName, rows) {
  const safeName = (sheetName || 'Sheet1').slice(0, 31).replace(/[\\/?*[\]:]/g, ' ')
  return [
    {
      name: '[Content_Types].xml',
      content:
        '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">' +
        '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>' +
        '<Default Extension="xml" ContentType="application/xml"/>' +
        '<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>' +
        '<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>' +
        '</Types>',
    },
    {
      name: '_rels/.rels',
      content:
        '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">' +
        '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>' +
        '</Relationships>',
    },
    {
      name: 'xl/workbook.xml',
      content:
        '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">' +
        `<sheets><sheet name="${xmlEscape(safeName)}" sheetId="1" r:id="rId1"/></sheets>` +
        '</workbook>',
    },
    {
      name: 'xl/_rels/workbook.xml.rels',
      content:
        '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">' +
        '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>' +
        '</Relationships>',
    },
    {
      name: 'xl/worksheets/sheet1.xml',
      content: buildSheetXml(rows),
    },
  ]
}

// ── สร้าง ZIP แบบ store (ไม่บีบอัด) จากรายการไฟล์ ───────────────────────────
function makeZip(files) {
  const encoded = files.map((f) => {
    const nameBytes = utf8(f.name)
    const dataBytes = utf8(f.content)
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

  // Local file headers + data
  encoded.forEach((f) => {
    localOffsets.push(offset)
    push(u32(0x04034b50)) // local file header signature
    push(u16(20)) // version needed
    push(u16(0)) // flags
    push(u16(0)) // method = store
    push(u16(0)) // mod time
    push(u16(0)) // mod date
    push(u32(f.crc)) // crc32
    push(u32(f.dataBytes.length)) // compressed size
    push(u32(f.dataBytes.length)) // uncompressed size
    push(u16(f.nameBytes.length)) // filename length
    push(u16(0)) // extra length
    push(f.nameBytes)
    push(f.dataBytes)
  })

  // Central directory
  const cdStart = offset
  encoded.forEach((f, i) => {
    push(u32(0x02014b50)) // central dir header signature
    push(u16(20)) // version made by
    push(u16(20)) // version needed
    push(u16(0)) // flags
    push(u16(0)) // method
    push(u16(0)) // mod time
    push(u16(0)) // mod date
    push(u32(f.crc)) // crc32
    push(u32(f.dataBytes.length)) // compressed size
    push(u32(f.dataBytes.length)) // uncompressed size
    push(u16(f.nameBytes.length)) // filename length
    push(u16(0)) // extra length
    push(u16(0)) // comment length
    push(u16(0)) // disk number start
    push(u16(0)) // internal attrs
    push(u32(0)) // external attrs
    push(u32(localOffsets[i])) // local header offset
    push(f.nameBytes)
  })
  const cdSize = offset - cdStart

  // End of central directory
  push(u32(0x06054b50)) // EOCD signature
  push(u16(0)) // disk number
  push(u16(0)) // disk with central dir
  push(u16(encoded.length)) // records on this disk
  push(u16(encoded.length)) // total records
  push(u32(cdSize)) // central dir size
  push(u32(cdStart)) // central dir offset
  push(u16(0)) // comment length

  return new Blob(chunks, {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  })
}

// สร้าง Blob .xlsx จากชื่อชีต + rows (array ของ array)
export function sheetToXlsxBlob(sheetName, rows) {
  return makeZip(buildParts(sheetName, rows))
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
