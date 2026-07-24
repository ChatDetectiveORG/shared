package pgfixture

import (
	"os"
	"strings"
	"testing"

	"github.com/go-pg/pg/v10"
)

const (
	envDatabaseURL = "CHAINTEST_DATABASE_URL"
	envMasterKey   = "MASTER_KEY"
	testMasterKey  = "01234567890123456789012345678901"

	// DefaultLocalDatabaseURL matches chatdetective-dev values-local.yaml when
	// PostgreSQL is port-forwarded: kubectl port-forward svc/chatdetective-dev-postgresql 5432:5432
	DefaultLocalDatabaseURL = "postgres://chatdetective:chatdetective@127.0.0.1:5432/chatdetective?sslmode=disable"
)

// EnsureCryptoEnv sets MASTER_KEY when missing so encryption helpers work in tests.
func EnsureCryptoEnv(t *testing.T) {
	t.Helper()
	if os.Getenv(envMasterKey) == "" {
		t.Setenv(envMasterKey, testMasterKey)
	}
}

// DatabaseURL returns CHAINTEST_DATABASE_URL, or the local dev stand URL when reachable.
func DatabaseURL(t *testing.T) string {
	t.Helper()
	EnsureCryptoEnv(t)
	if url := strings.TrimSpace(os.Getenv(envDatabaseURL)); url != "" {
		return url
	}
	if canConnect(DefaultLocalDatabaseURL) {
		return DefaultLocalDatabaseURL
	}
	t.Skipf("%s is not set and local dev postgres is unavailable at %s; "+
		"port-forward with: kubectl port-forward svc/chatdetective-dev-postgresql 5432:5432",
		envDatabaseURL, DefaultLocalDatabaseURL)
	return ""
}

func canConnect(url string) bool {
	opt, err := pg.ParseURL(url)
	if err != nil {
		return false
	}
	db := pg.Connect(opt)
	defer db.Close()
	return db.Ping(db.Context()) == nil
}
