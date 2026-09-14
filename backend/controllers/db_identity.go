package controllers

import (
	"log"

	"gorm.io/gorm"
)

// tableNameOf resolves the real table name for a model so the helpers below
// never have to hard-code (and get wrong) GORM's pluralisation.
func tableNameOf(db *gorm.DB, model interface{}) string {
	if db == nil || model == nil {
		return ""
	}
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		log.Println("resolve table name:", err)
		return ""
	}
	return stmt.Schema.Table
}

// ResetIdentityIfEmpty rolls a table's primary-key sequence back to 1, but only
// when the table has no rows left.
//
// Why this exists: id is fed by a database sequence, and a sequence never goes
// backwards on its own. So "clear everything, then upload the same file again"
// used to hand out fresh ids every time (1,2,3 -> 4,5,6 -> 7,8,9 ...) even
// though the data was identical. Rewinding the sequence on an empty table makes
// the next upload start at 1 again.
//
// The emptiness guard matters: if any row survives we must leave the sequence
// alone, or the next insert would collide with an id that is still in use.
func ResetIdentityIfEmpty(db *gorm.DB, model interface{}) {
	table := tableNameOf(db, model)
	if table == "" {
		return
	}

	var remaining int64
	if err := db.Table(table).Count(&remaining).Error; err != nil {
		log.Println("reset identity: count", table, ":", err)
		return
	}
	if remaining > 0 {
		return
	}

	switch db.Dialector.Name() {
	case "postgres":
		// is_called = false means the next value handed out is 1 itself.
		if err := db.Exec(
			`SELECT setval(pg_get_serial_sequence(?, 'id'), 1, false)`, table,
		).Error; err != nil {
			log.Println("reset identity:", table, ":", err)
		}
	case "sqlite", "sqlite3":
		// Error ignored on purpose: sqlite_sequence only exists once an
		// AUTOINCREMENT table has actually been written to.
		db.Exec(`DELETE FROM sqlite_sequence WHERE name = ?`, table)
	}
}

// SyncIdentityToMax pushes the sequence just past the largest id currently in
// the table. Needed after any write that inserts explicit ids, because those
// inserts bypass the sequence and would otherwise let a later insert reuse an
// id that is already taken.
func SyncIdentityToMax(db *gorm.DB, model interface{}) {
	if db == nil || db.Dialector.Name() != "postgres" {
		return
	}
	table := tableNameOf(db, model)
	if table == "" {
		return
	}
	if err := db.Exec(
		`SELECT setval(pg_get_serial_sequence(?, 'id'),
		               GREATEST(COALESCE((SELECT MAX(id) FROM "`+table+`"), 0), 1))`,
		table,
	).Error; err != nil {
		log.Println("sync identity:", table, ":", err)
	}
}
