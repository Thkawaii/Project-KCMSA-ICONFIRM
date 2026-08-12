package models

import "time"

type User struct {
	ID uint `gorm:"primaryKey"`

	RoleName string `gorm:"size:50"`

	// Username ไม่ unique แล้ว — หลายคนในแผนกเดียวใช้ username ร่วมกันได้
	// (เช่น wh@kobelco.com ใช้ร่วมกันทั้งแผนก WH) แล้วแยกคนด้วย "รหัสผ่านเฉพาะคน"
	// ตอน login ระบบจะ match (username + password) เพื่อรู้ว่าเป็นพนักงานคนไหน
	// -> ใช้บันทึก "Checked By" ตอนสแกนได้ถูกคน
	Username string `gorm:"size:100;index"`

	Password string `gorm:"size:255"`

	Status string `gorm:"size:20"`

	Name string `gorm:"size:100"`

	CreatedAt time.Time
}
