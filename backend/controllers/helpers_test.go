package controllers

import "testing"

// ─────────────────────────────────────────────────────────────────────────────
// ทดสอบฟังก์ชันช่วย (helpers) ที่เป็น logic ล้วน — สแกน/normalize ค่าจาก Excel/บาร์โค้ด
// ─────────────────────────────────────────────────────────────────────────────

func TestLooks12Digit(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"878250022802", true},      // 12 หลัก
		{"1234567890", true},        // 10 หลัก (ขอบล่าง)
		{"123456789012345", true},   // 15 หลัก (ขอบบน)
		{"  878250022802  ", true},  // มีช่องว่างหน้า/หลัง
		{"123456789", false},        // 9 หลัก สั้นไป
		{"1234567890123456", false}, // 16 หลัก ยาวไป
		{"87825002280A", false},     // มีตัวอักษร
		{"8782-5002-2802", false},   // มีขีด
		{"", false},
	}
	for _, tc := range cases {
		if got := looks12Digit(tc.in); got != tc.want {
			t.Errorf("looks12Digit(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeDigitCell(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"878180022402.0", "878180022402"},   // จุดทศนิยมศูนย์ล้วน 1 ตัว
		{"878180022402.00", "878180022402"},  // ศูนย์หลายตัว
		{"878180022402.000", "878180022402"}, // ศูนย์สามตัว
		{"  878180022402  ", "878180022402"}, // trim
		{"878180022402", "878180022402"},     // ปกติไม่แตะ
		{"878180022402.5", "878180022402.5"}, // มีเศษไม่ใช่ศูนย์ → คงไว้
		{"", ""},
	}
	for _, tc := range cases {
		if got := normalizeDigitCell(tc.in); got != tc.want {
			t.Errorf("normalizeDigitCell(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	// scientific notation → ต้องขยายกลับเป็นเลขเต็ม
	if got := normalizeDigitCell("1.23E5"); got != "123000" {
		t.Errorf("normalizeDigitCell(1.23E5) = %q, want 123000", got)
	}
}

func TestNormalizeCodeValue(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"YN22E00849FA", "YN22E00849FA"},
		{"yn22e00849fa", "YN22E00849FA"},   // ทำเป็นตัวใหญ่
		{"YN22-E008 49FA", "YN22E00849FA"}, // ตัดขีด/ช่องว่าง
		{" MC-LC14405563 ", "MCLC14405563"},
		{`="878250022802"`, "878250022802"}, // unwrap Excel ="..."
		{"", ""},
	}
	for _, tc := range cases {
		if got := NormalizeCodeValue(tc.in); got != tc.want {
			t.Errorf("NormalizeCodeValue(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestUnwrapExcelText(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`="878250022802"`, "878250022802"},
		{`="ABC"`, "ABC"},
		{"plain", "plain"},
		{`"quoted"`, `"quoted"`}, // ไม่มี =" นำหน้า → ไม่แตะ
		{"", ""},
	}
	for _, tc := range cases {
		if got := unwrapExcelText(tc.in); got != tc.want {
			t.Errorf("unwrapExcelText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAtoiSafe(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"12", 12},
		{"12.0", 12}, // Excel เติม .0
		{"  7 ", 7},
		{"abc", 0},
		{"", 0},
	}
	for _, tc := range cases {
		if got := atoiSafe(tc.in); got != tc.want {
			t.Errorf("atoiSafe(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestPickField(t *testing.T) {
	m := map[string]string{
		"Machine": "LX10400690",
		"empty":   "   ",
		"Product": "SK75",
	}
	// คืนค่าที่ไม่ว่างตัวแรกตามลำดับ key ที่ลอง
	if got := pickField(m, "missing", "empty", "Machine"); got != "LX10400690" {
		t.Errorf("pickField priority = %q, want LX10400690", got)
	}
	if got := pickField(m, "missing1", "missing2"); got != "" {
		t.Errorf("pickField none = %q, want empty", got)
	}
}

func TestMachineFromRow(t *testing.T) {
	// รองรับทั้งคอลัมน์มาตรฐานและคอลัมน์นอกสเปกที่ขึ้นต้นด้วย "[+] "
	if got := machineFromRow(map[string]string{"Machine No": "A1"}); got != "A1" {
		t.Errorf("machineFromRow standard = %q, want A1", got)
	}
	if got := machineFromRow(map[string]string{"[+] Machine": "B2"}); got != "B2" {
		t.Errorf("machineFromRow plus-prefixed = %q, want B2", got)
	}
	if got := machineFromRow(map[string]string{"nothing": "x"}); got != "" {
		t.Errorf("machineFromRow none = %q, want empty", got)
	}
}
