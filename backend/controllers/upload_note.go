package controllers

import (
	"sort"
	"strings"
	"unicode"
)

const uploadNoteLabel = "Note"

var noteDeleteWords = map[string]bool{
	"delete":   true,
	"ลบ":       true,
	"ลบข้อมูล": true,
}

func IsDeleteNote(note string) bool {
	s := strings.ToLower(strings.TrimSpace(note))
	if s == "" {
		return false
	}
	if noteDeleteWords[s] {
		return true
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	})
	return len(parts) > 0 && noteDeleteWords[parts[0]]
}

func sameOptionalCode(fileValue, dbValue string) bool {
	f := strings.TrimSpace(fileValue)
	d := strings.TrimSpace(dbValue)
	if f == "" || d == "" {
		return true
	}
	return NormalizeCodeValue(f) == NormalizeCodeValue(d)
}

func sameOptionalText(fileValue, dbValue string) bool {
	f := strings.TrimSpace(fileValue)
	d := strings.TrimSpace(dbValue)
	if f == "" || d == "" {
		return true
	}
	return strings.EqualFold(f, d)
}

func uploadDataSignature(data map[string]string, only map[string]bool) string {
	keys := make([]string, 0, len(data))
	for k, v := range data {
		if k == uploadNoteLabel || strings.TrimSpace(v) == "" {
			continue
		}
		if only != nil && !only[k] {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(0)
		b.WriteString(strings.TrimSpace(data[k]))
		b.WriteByte(1)
	}
	return b.String()
}
