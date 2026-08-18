package models

import "testing"

// ─────────────────────────────────────────────────────────────────────────────
// ทดสอบฟังก์ชันบริสุทธิ์ (pure functions) ของ IT Controller connectivity
// ไม่ต้องต่อฐานข้อมูล — รันได้ทันทีด้วย `go test ./models/...`
// ─────────────────────────────────────────────────────────────────────────────

func TestClassifyConnectivity(t *testing.T) {
	cases := []struct {
		name     string
		partName string
		model    string
		want     string
	}{
		{"iridium satellite", "Q4000 IRIDIUM IT CONTROLLER", "JRN-260K", ConnSatelliteIrid},
		{"satellite keyword", "Satellite Terminal", "", ConnSatelliteIrid},
		{"sat abbrev", "SAT UNIT", "", ConnSatelliteIrid},
		{"4g high speed", "IT Controller 4G HIGH", "", ConnMobile4GHigh},
		{"hs abbrev counts as high", "MODULE HS", "", ConnMobile4GHigh},
		{"4g normal", "Mobile 4G Normal", "", ConnMobile4GNormal},
		{"lte maps to normal", "LTE MODULE", "", ConnMobile4GNormal},
		{"plain mobile", "MOBILE UNIT", "", ConnMobile4GNormal},
		{"unknown returns empty", "Random Widget", "ABC-1", ""},
		{"empty input", "", "", ""},
		// ลำดับความสำคัญ: iridium/satellite มาก่อน high มาก่อน 4g/normal
		{"iridium wins over 4g", "4G IRIDIUM", "", ConnSatelliteIrid},
		{"high wins over normal", "4G HIGH NORMAL", "", ConnMobile4GHigh},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyConnectivity(tc.partName, tc.model); got != tc.want {
				t.Errorf("ClassifyConnectivity(%q,%q) = %q, want %q", tc.partName, tc.model, got, tc.want)
			}
		})
	}
}

func TestNormalizeConnectivity(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"Satellite", ConnSatelliteIrid},
		{"iridium", ConnSatelliteIrid},
		{"4G High", ConnMobile4GHigh},
		{"4g normal", ConnMobile4GNormal},
		{"  mobile  ", ConnMobile4GNormal},
		// ค่ารหัสมาตรฐานที่ป้อนตรง ๆ ต้องคืนค่าเดิม
		{ConnMobile4GNormal, ConnMobile4GNormal},
		{ConnMobile4GHigh, ConnMobile4GHigh},
		{ConnSatelliteIrid, ConnSatelliteIrid},
		{"garbage", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := NormalizeConnectivity(tc.raw); got != tc.want {
			t.Errorf("NormalizeConnectivity(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

// กันการเผลอแก้ค่าคงที่สถานะ (ถ้าเปลี่ยนค่า test นี้จะเตือน)
func TestStatusConstantsStable(t *testing.T) {
	pairs := map[string]string{
		MFGStatusMatched:    "MATCHED",
		MFGStatusNotMatched: "NOT_MATCHED",
		MFGStatusDuplicate:  "DUPLICATE",
	}
	for got, want := range pairs {
		if got != want {
			t.Errorf("MFG status constant = %q, want %q", got, want)
		}
	}
}
