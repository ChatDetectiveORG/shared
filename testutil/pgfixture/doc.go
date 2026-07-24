// Package pgfixture seeds PostgreSQL state for service chain tests.
//
// Chain tests skip when CHAINTEST_DATABASE_URL is unset. Set MASTER_KEY (32 bytes)
// or let EnsureCryptoEnv install a deterministic test key.
package pgfixture
