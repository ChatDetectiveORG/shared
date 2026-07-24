package pgfixture

import (
	"testing"

	models "github.com/ChatDetectiveORG/shared/postgresModels"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

var schemaModels = []interface{}{
	(*models.Message)(nil),
	(*models.Telegramuser)(nil),
	(*models.UserSettings)(nil),
	(*models.Admin)(nil),
	(*models.MessageVersion)(nil),
	(*models.UserLevels)(nil),
}

// Open connects to CHAINTEST_DATABASE_URL and ensures schema exists.
func Open(t *testing.T) *pg.DB {
	t.Helper()
	opt, err := pg.ParseURL(DatabaseURL(t))
	if err != nil {
		t.Fatalf("pgfixture: parse database url: %v", err)
	}
	db := pg.Connect(opt)
	EnsureSchema(t, db)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// EnsureSchema creates tables required by chain tests.
func EnsureSchema(t *testing.T, db *pg.DB) {
	t.Helper()
	for _, model := range schemaModels {
		if err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		}); err != nil {
			t.Fatalf("pgfixture: create table for %T: %v", model, err)
		}
	}
}

// Reset deletes rows from chain-test tables between scenarios.
func Reset(t *testing.T, db *pg.DB) {
	t.Helper()
	for _, model := range schemaModels {
		if _, err := db.Model(model).Where("TRUE").Delete(); err != nil {
			t.Fatalf("pgfixture: reset %T: %v", model, err)
		}
	}
}
