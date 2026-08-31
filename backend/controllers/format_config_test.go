package controllers

import (
	"testing"

	"iconfirm/models"
)

// เพิ่ม alias ของหมายเลขเครื่องต้องทำได้ ถ้าเครื่องนั้นมีอยู่จริงในระบบ
//
// หมายเลขเครื่อง (LX10400691) ไม่ได้อยู่ใน master_data แต่อยู่ในแผนประกอบ /
// รายการ MFG / ใบอนุญาตส่งออก เดิม registry มองแค่ master_data
// จึงตอบว่า "ไม่พบ Old (ค่าเดิม) ในระบบ" ทั้งที่มีอยู่
func TestRegistryIndexKnowsMachineNumbers(t *testing.T) {
	db := newTestDB(t)

	// เครื่องอยู่ในแผนประกอบเท่านั้น ไม่ได้อยู่ใน master_data
	seedPlan(t, db, "LX10400691", "878250022802", "Vietnam")

	if !oldValueExistsInRegistry("machine", "LX10400691") {
		t.Error("หมายเลขเครื่องจากแผนประกอบต้องถือว่ามีอยู่ในระบบ")
	}
	if oldValueExistsInRegistry("machine", "LX99999999") {
		t.Error("เครื่องที่ไม่มีอยู่จริงต้องไม่ผ่าน")
	}
}

func TestRegistryIndexKnowsMachineFromMFGAndExport(t *testing.T) {
	db := newTestDB(t)

	db.Create(&models.MFGAssembly{MachineNo: "LX10400690", ITControllerNo: "878250022801"})
	db.Create(&models.ExportLicenseItem{SerialNumber: "878250022803", MachineNo: "LX10400692"})

	for _, mc := range []string{"LX10400690", "LX10400692"} {
		if !oldValueExistsInRegistry("machine", mc) {
			t.Errorf("%s ต้องถือว่ามีอยู่ในระบบ", mc)
		}
	}
}
