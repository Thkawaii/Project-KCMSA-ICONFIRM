package controllers

import (
	"strings"

	"iconfirm/config"
	"iconfirm/models"

	"github.com/gin-gonic/gin"
)

// MachineProfile คือสเปกเครื่องหนึ่งคัน ประกอบจากข้อมูลที่อัปโหลดจริง
// (Planning + Assembly + Engine + ทะเบียน Master Data)
//
// เดิมหน้า QA ดึงจากตาราง machine_specs ซึ่งไม่มีข้อมูลเลย หน้าจอจึงขึ้น
// "ไม่พบข้อมูลเครื่องนี้ในระบบ" ตลอด — ย้ายมาอ่านจากแหล่งที่มีข้อมูลจริง
// ชื่อฟิลด์ยังคงเดิมทุกตัว เพื่อให้หน้า QA ใช้ต่อได้โดยไม่ต้องแก้โครงสร้าง
type MachineProfile struct {
	MachineNo string `json:"MachineNo"`

	Spec1    string `json:"Spec1"`
	Spec2    string `json:"Spec2"`
	KCMOrder string `json:"KCMOrder"`
	BaseSpec string `json:"BaseSpec"`

	Boom     string `json:"Boom"`
	BoomNo   string `json:"BoomNo"`
	BoomName string `json:"BoomName"`

	Arm     string `json:"Arm"`
	ArmNo   string `json:"ArmNo"`
	ArmName string `json:"ArmName"`

	FrontATT string `json:"FrontATT"`
	BucketNo string `json:"BucketNo"`

	CountryName string `json:"CountryName"`
	OtherPiping string `json:"OtherPiping"`
	DigNavi     string `json:"DigNavi"`
	CabGuard    string `json:"CabGuard"`

	Engine         string `json:"Engine"`
	EngineHistory  string `json:"EngineHistory"`
	EngineStartKey string `json:"EngineStartKey"`

	Radio       string `json:"Radio"`
	OtherOption string `json:"OtherOption"`

	CWNo     string `json:"CWNo"`
	CWName   string `json:"CWName"`
	CWWeight string `json:"CWWeight"`

	Shoe string `json:"Shoe"`
	Seat string `json:"Seat"`

	ITDevice       string `json:"ITDevice"`
	ITController   string `json:"ITController"`
	ITControllerSN string `json:"ITControllerSN"`
	ITControllerNo string `json:"ITControllerNo"`

	ControlValve string `json:"ControlValve"`
	SwingMotor   string `json:"SwingMotor"`
	MotorPropel  string `json:"MotorPropel"`
	PumpAssyHyd  string `json:"PumpAssyHyd"`

	HydOil string `json:"HydOil"`

	AssemblyPartsNumber string `json:"AssemblyPartsNumber"`
	AssemblyPartsName   string `json:"AssemblyPartsName"`
}

func buildMachineProfile(machineNo string, plan map[string]string) MachineProfile {
	p := MachineProfile{MachineNo: machineNo}

	if plan == nil {
		return p
	}

	get := func(keys ...string) string { return planValue(plan, keys...) }

	p.Spec1 = get("Product Spec 1", "Spec Code", "Spec(1)")
	p.Spec2 = get("Product Spec 2", "Specification Detail", "Spec(2)")
	p.KCMOrder = get("KCM Order", "Order No")
	p.BaseSpec = get("Base machine spec.", "Base spec", "Specification Detail")

	p.Boom = get("Boom")
	p.BoomNo = get("Boom no", "Boom No")
	p.BoomName = get("Boom name", "Boom Name")

	p.Arm = get("Arm")
	p.ArmNo = get("Arm no", "Arm No")
	p.ArmName = get("Arm name", "Arm Name")

	p.FrontATT = get("Front ATT")
	p.BucketNo = get("Bucket no", "Bucket No")

	p.CountryName = plannedCountryOf(plan)
	p.OtherPiping = get("Other piping")
	p.DigNavi = get("DigNavi")
	p.CabGuard = get("Cab guard")

	p.EngineStartKey = get("Engine start key")
	p.Radio = get("Radio")
	p.OtherOption = get("Other option")

	p.CWNo = get("CW no", "CW No")
	p.CWName = get("CW name", "CW Name")
	p.CWWeight = get("Counter weight", "CW weight")

	p.Shoe = get("Shoe")
	p.Seat = get("Seat")

	p.ITDevice = plannedDeviceOf(plan)
	p.ITControllerNo = PlannedITCOf(plan)

	p.ControlValve = get("Control Valve No")
	p.SwingMotor = get("Swing Motor No")
	p.MotorPropel = get("Motor Propel No")
	p.PumpAssyHyd = get("Pump Assy HYD No")

	p.HydOil = get("Cold region spec(HYD oil)", "HYD oil")

	p.AssemblyPartsNumber = get("Assembly_Parts_Number", "Assembly Parts Number")
	p.AssemblyPartsName = get("Assembly_Parts_Name", "Assembly Parts Name")

	return p
}

// fillEngineFromUpload เติมข้อมูลเครื่องยนต์จากไฟล์ Engine ที่อัปโหลดแยกไว้
func fillEngineFromUpload(p *MachineProfile) {
	if p.MachineNo == "" {
		return
	}
	for _, row := range loadUploadRows(models.DatasetEngine) {
		if strings.TrimSpace(pickField(row, "Machine No", "Machine")) != p.MachineNo {
			continue
		}
		if p.Engine == "" {
			p.Engine = strings.TrimSpace(pickField(row, "ENGINE", "Engine"))
		}
		if p.EngineHistory == "" {
			p.EngineHistory = strings.TrimSpace(pickField(row, "History", "Engine History"))
		}
		return
	}
}

// fillITControllerFromMaster เติม P/N และ S/N ของ IT Controller จากทะเบียน Master Data
func fillITControllerFromMaster(p *MachineProfile) {
	if p.ITControllerNo == "" {
		return
	}
	var m models.MasterData
	if err := config.DB.Where("it_controller_no = ?", p.ITControllerNo).
		First(&m).Error; err != nil {
		return
	}
	p.ITController = m.PartNo
	p.ITControllerSN = m.SerialNo
	if p.Spec1 == "" {
		p.Spec1 = m.SpecCode
	}
}

func GetMachineProfile(c *gin.Context) {
	machineNo := strings.TrimSpace(c.Param("machineNo"))
	if machineNo == "" {
		c.JSON(400, gin.H{"message": "ต้องระบุหมายเลขเครื่อง"})
		return
	}

	plan := planForMachine(machineNo)
	if plan == nil {
		c.JSON(404, gin.H{
			"message": "ไม่พบข้อมูลเครื่อง " + machineNo +
				" — กรุณาอัปโหลดไฟล์ Planning แล้วกดปั๊มตาราง Assembly ก่อน",
		})
		return
	}

	profile := buildMachineProfile(machineNo, plan)
	fillEngineFromUpload(&profile)
	fillITControllerFromMaster(&profile)

	c.JSON(200, profile)
}
