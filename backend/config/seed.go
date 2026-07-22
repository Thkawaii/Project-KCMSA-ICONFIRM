package config

import "iconfirm/models"

func SeedData() {

	var count int64

	DB.Model(&models.User{}).Count(&count)

	if count > 0 {
		return
	}

	users := []models.User{
		{
			RoleName: "QA",
			Username: "qa@kobelco.com",
			Password: "qa.kobelco",
			Status:   "Active",
			Name:     "QA User",
		},
		{
			RoleName: "WH",
			Username: "wh@kobelco.com",
			Password: "wh.kobelco",
			Status:   "Active",
			Name:     "Warehouse User",
		},
		{
			RoleName: "TSF",
			Username: "mfg@kobelco.com",
			Password: "mfg.kobelco",
			Status:   "Active",
			Name:     "TSF Operator",
		},
		{
			RoleName: "UPLOAD",
			Username: "uploadview@kobelco.com",
			Password: "uploadview.kobelco",
			Status:   "Active",
			Name:     "Upload View",
		},
	}

	DB.Create(&users)

	// สำคัญ: หลัง DB.Create(&users) แล้ว GORM จะเติม ID จริงกลับเข้ามาใน
	// slice users ให้อัตโนมัติ (users[0].ID, users[1].ID, ...) เอาไว้ผูกกับ
	// UserID ของตารางอื่น — ถ้าไม่ทำแบบนี้ UserID จะเป็น 0 ตาม zero-value
	// ของ Go แล้วชน FK constraint (fk_master_data_user, fk_qas_user ฯลฯ)
	// เหมือนที่เจอ error อยู่ตอนนี้
	qaUserID := users[0].ID
	whUserID := users[1].ID
	tsfUserID := users[2].ID

	master := []models.MasterData{
		{
			Name:     "IT Controller",
			PartNo:   "YN02P00133F2G1",
			SerialNo: "J05ETG63544",
			SpecCode: "SPEC001",
			UserID:   whUserID,
		},
		{
			Name:     "Control Valve",
			PartNo:   "CV001",
			SerialNo: "SN10001",
			SpecCode: "SPEC002",
			UserID:   whUserID,
		},
	}

	DB.Create(&master)

	warehouse := []models.Warehouse{
		{
			Warehouse:        "WH01",
			OrderNo:          "SO2026001",
			WorkOrder:        "WO001",
			StockOutNo:       "ST001",
			PartNo:           "YN02P00133F2G1",
			PartName:         "IT Controller",
			AssemblyPartNo:   "ASM001",
			AssemblyPartName: "Controller Assembly",
			RemainQty:        5,
			StandardCost:     25000,
			Shelf1:           "A01",
			Shelf2:           "B01",
			MachineModel:     "SK200",
			FinalColor:       "Orange",
			UserID:           whUserID,
		},
	}

	DB.Create(&warehouse)

	tsf := []models.TSFOperator{
		{
			SerialNumber:     "J05ETG63544",
			ActualPartNo:     "YN02P00133F2G1",
			ActualSpecCode:   "SPEC001",
			ValidationStatus: "PASS",
			FileName:         "it_controller.jpg",
			ScannedBy:        "operator",
			UserID:           tsfUserID,
		},
	}

	DB.Create(&tsf)

	qa := []models.QA{
		{
			ExpectedPartNo: "YN02P00133F2G1",
			ActualPartNo:   "YN02P00133F2G1",
			ExpectedSpec:   "SPEC001",
			ActualSpec:     "SPEC001",
			Result:         "PASS",
			UserID:         qaUserID,
		},
	}

	DB.Create(&qa)
}