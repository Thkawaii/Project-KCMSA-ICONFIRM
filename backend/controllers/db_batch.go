package controllers

import (
	"strconv"

	"gorm.io/gorm"
)

const (
	dbInListChunk = 10000

	dbInsertBatch = 500

	maxUploadProblems = 300
)

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
