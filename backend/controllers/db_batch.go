package controllers

import (
	"strconv"

	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// ตัวช่วยแบ่งคำสั่ง SQL เป็นก้อน ๆ สำหรับไฟล์ใหญ่
//
// PostgreSQL (โปรโตคอล extended ที่ pgx ใช้) รับ "พารามิเตอร์" ได้สูงสุด 65,535 ตัวต่อ 1 คำสั่ง
// ถ้าเกินจะเจอ error:  extended protocol limited to 65535 parameters
//
// ที่ผ่านมาเกิดได้ 2 จุด
//   1. INSERT ทีเดียวทั้งไฟล์ — ใบอนุญาตส่งออกมี ~23 คอลัมน์ต่อแถว
//      จึงพังตั้งแต่ประมาณ 2,850 แถวขึ้นไป (23 × 2,850 ≈ 65,535)
//   2. WHERE ... IN ? ที่ส่งค่าทั้งไฟล์ไปในคำสั่งเดียว — พังเมื่อเกิน 65,535 ค่า
//
// ไฟล์นี้จึงแบ่งทั้งสองแบบให้อยู่ใต้เพดานเสมอ ไม่ว่าไฟล์จะมีกี่แถวก็ตาม
// ---------------------------------------------------------------------------

const (
	// dbInListChunk จำนวนค่าสูงสุดใน WHERE ... IN ? ต่อ 1 คำสั่ง
	dbInListChunk = 10000

	// dbInsertBatch จำนวนแถวต่อ 1 คำสั่ง INSERT
	// 500 แถว × ~25 คอลัมน์ ≈ 12,500 พารามิเตอร์ — ยังห่างเพดานมาก เผื่อเพิ่มคอลัมน์ในอนาคต
	dbInsertBatch = 500

	// maxUploadProblems จำนวนข้อความแจ้งปัญหาสูงสุดที่ส่งกลับไปแสดงบนหน้าจอ
	// ไฟล์หลักหมื่นแถวอาจมีแถวซ้ำเป็นพัน — ส่งทั้งหมดจะทำให้หน้าเว็บค้าง
	maxUploadProblems = 300
)

// chunkSlice แบ่ง slice เป็นก้อนละไม่เกิน size ตัว (ก้อนสุดท้ายอาจสั้นกว่า)
func chunkSlice[T any](list []T, size int) [][]T {
	if size <= 0 {
		size = 1
	}
	if len(list) == 0 {
		return nil
	}
	out := make([][]T, 0, (len(list)+size-1)/size)
	for start := 0; start < len(list); start += size {
		end := start + size
		if end > len(list) {
			end = len(list)
		}
		out = append(out, list[start:end:end])
	}
	return out
}

// findWhereInChunks = db.Where("<column> IN ?", values).Find(out) แต่แบ่งยิงทีละก้อน
// ผลลัพธ์ทุกก้อนต่อกันไว้ใน out
func findWhereInChunks[T any, V any](db *gorm.DB, column string, values []V, out *[]T) error {
	for _, part := range chunkSlice(values, dbInListChunk) {
		var batch []T
		if err := db.Where(column+" IN ?", part).Find(&batch).Error; err != nil {
			return err
		}
		*out = append(*out, batch...)
	}
	return nil
}

// capProblems ตัดรายการแจ้งปัญหาให้เหลือไม่เกิน maxUploadProblems บรรทัด
// แล้วต่อท้ายด้วยจำนวนที่เหลือ เพื่อไม่ให้ response ใหญ่จนหน้าเว็บค้าง
func capProblems(problems []string) []string {
	if len(problems) <= maxUploadProblems {
		return problems
	}
	rest := len(problems) - maxUploadProblems
	out := make([]string, 0, maxUploadProblems+1)
	out = append(out, problems[:maxUploadProblems]...)
	out = append(out, "… และอีก "+strconv.Itoa(rest)+" รายการ")
	return out
}

// clampRunes ตัดสตริงให้ยาวไม่เกิน n ตัวอักษร (นับแบบ rune เหมือน varchar(n) ของ PostgreSQL)
func clampRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}
