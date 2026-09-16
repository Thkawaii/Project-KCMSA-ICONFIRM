package controllers

import "testing"

func TestIsDeleteNote(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Delete", true},
		{"delete", true},
		{" DELETE ", true},
		{"ลบ", true},
		{"ลบข้อมูล", true},
		{"Delete - P/N ผิด", true},
		{"ลบ เพราะพิมพ์ผิด", true},
		{"", false},
		{"ห้ามลบ", false},
		{"ลบมุม", false},
		{"deleted", false},
		{"ok", false},
	}
	for _, tc := range cases {
		if got := IsDeleteNote(tc.in); got != tc.want {
			t.Errorf("IsDeleteNote(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestUploadDataSignatureIgnoresNote(t *testing.T) {
	a := map[string]string{"Machine": "LX1", "Note": "", "Line": "1"}
	b := map[string]string{"Machine": "LX1", "Note": "Delete", "Line": "1", "Empty": ""}
	if uploadDataSignature(a, nil) != uploadDataSignature(b, nil) {
		t.Error("signature ต้องไม่ขึ้นกับ Note และค่าว่าง")
	}
	c := map[string]string{"Machine": "LX2", "Line": "1"}
	if uploadDataSignature(a, nil) == uploadDataSignature(c, nil) {
		t.Error("ข้อมูลต่างกันต้องได้ signature ต่างกัน")
	}
}
