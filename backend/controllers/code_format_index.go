package controllers

import (
	"strings"

	"iconfirm/config"
	"iconfirm/models"
)

// codeFormatIndex คือสำเนาในหน่วยความจำของตาราง Change Format Part
// ให้ผลเหมือน resolveByKind("", …) / CurrentCodeOf / CodeVariants / RetiredCodeReplacement ทุกประการ
// แต่โหลดตารางครั้งเดียวต่อหนึ่ง request
//
// ฟังก์ชันตัวจริงใน format_config.go ค้นฐานข้อมูลทุกครั้งที่เรียก
// ซึ่งหน้า QA ต้องแปลงรหัสเป็นพัน ๆ ครั้ง (แผน × ชนิดพาร์ท × ผลสแกน WH × แถว MFG)
type codeFormatIndex struct {
	// oldByNew: New (ค่าใหม่) → Old (ค่าเดิม) — เอาแถวแรก (id น้อยสุด) เหมือน findCodeAliasByFromCode
	oldByNew map[string]string
	// newByOld: Old (ค่าเดิม) → New (ค่าใหม่) — เอาแถวล่าสุด (id มากสุด) เหมือน findCodeAliasByToOld
	newByOld map[string]string

	currentCache  map[string]string
	variantsCache map[string][]string
}

func loadCodeFormatIndex() *codeFormatIndex {
	x := &codeFormatIndex{
		oldByNew:      map[string]string{},
		newByOld:      map[string]string{},
		currentCache:  map[string]string{},
		variantsCache: map[string][]string{},
	}

	var rows []models.CodeAlias
	config.DB.Order("id asc").Find(&rows)
	for _, r := range rows {
		from := strings.TrimSpace(r.FromCode)
		to := strings.TrimSpace(r.ToOld)

		if nf := NormalizeCodeValue(from); nf != "" {
			if _, ok := x.oldByNew[nf]; !ok {
				x.oldByNew[nf] = to
			}
		}
		if nt := NormalizeCodeValue(to); nt != "" {
			x.newByOld[nt] = from
		}
	}
	return x
}

// old แปลงรหัสใด ๆ เป็นค่าเดิมในระบบ (เท่ากับ resolveByKind("", raw))
func (x *codeFormatIndex) old(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if v, ok := x.oldByNew[NormalizeCodeValue(raw)]; ok && v != "" {
		return v
	}
	return raw
}

// current คืนรหัส "รูปแบบที่ใช้อยู่ตอนนี้" (เท่ากับ CurrentCodeOf)
func (x *codeFormatIndex) current(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if hit, ok := x.currentCache[raw]; ok {
		return hit
	}
	out := raw
	if v, ok := x.newByOld[NormalizeCodeValue(x.old(raw))]; ok && v != "" {
		out = v
	}
	x.currentCache[raw] = out
	return out
}

// variants คืนรหัสทุกรูปแบบของค่าเดียวกัน (เท่ากับ CodeVariants)
func (x *codeFormatIndex) variants(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if hit, ok := x.variantsCache[raw]; ok {
		return hit
	}
	out := dedupeCodes(x.current(raw), x.old(raw), raw)
	x.variantsCache[raw] = out
	return out
}

// scanKeys คืนคีย์ qaScanKey ของทุกรูปแบบ (ไม่ซ้ำ ไม่ว่าง) ใช้ทำดัชนีจับคู่ข้ามรูปแบบ
func (x *codeFormatIndex) scanKeys(raw string) []string {
	vs := x.variants(raw)
	out := make([]string, 0, len(vs))
	seen := map[string]bool{}
	for _, v := range vs {
		k := qaScanKey(v)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return out
}

// formerCodes คืนรูปแบบอื่นที่ไม่ใช่รูปแบบปัจจุบัน (ใช้ให้ค้นหาด้วยรหัสเก่าได้ / แสดงว่าเดิมคืออะไร)
func (x *codeFormatIndex) formerCodes(raw string) []string {
	cur := x.current(raw)
	var out []string
	for _, v := range x.variants(raw) {
		if !SameCode(v, cur) {
			out = append(out, v)
		}
	}
	return out
}
