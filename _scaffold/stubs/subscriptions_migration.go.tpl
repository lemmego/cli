package migrations

import (
	"database/sql"
	"fmt"

	"github.com/lemmego/migration"
)

func init() {
	migration.GetMigrator().AddMigration(&migration.Migration{
		Version: "20261111000000",
		Up:      mig_20261111000000_create_subscriptions_table_up,
		Down:    mig_20261111000000_create_subscriptions_table_down,
	})
}

// Two tables for payments: what is subscribed, and which events have been
// seen.
//
// Raw SQL rather than the schema builder, because both tables need something
// it cannot express — a unique constraint that ON CONFLICT can name as its
// target. Generated for {{.SQLDriver}}; the module is offered on postgres and
// sqlite only, because the store's upserts are that syntax.
//
// payment_events is the idempotency record and nothing else. Stripe delivers
// every event at least twice, and the unique index on event_id is the entire
// defence: the insert either succeeds, which means this delivery is the
// first, or it conflicts, which means it has been handled. A SELECT followed
// by an INSERT would have a window exactly wide enough for two concurrent
// deliveries to both decide they were first.
//
// subscriptions.event_at is what makes out-of-order delivery safe. Stripe
// makes no ordering promise, so the write that applies an event is
// conditional on being newer than what the row already holds — see
// internal/billing.
func mig_20261111000000_create_subscriptions_table_up(tx *sql.Tx) error {
	statements := []string{
{{- if eq .SQLDriver "postgres"}}
		`CREATE TABLE subscriptions (
			id                   BIGSERIAL PRIMARY KEY,
			subscription_id      TEXT        NOT NULL UNIQUE,
			customer_id          TEXT        NOT NULL,
			status               TEXT        NOT NULL,
			price_id             TEXT        NOT NULL DEFAULT '',
			current_period_end   TIMESTAMPTZ,
			cancel_at_period_end BOOLEAN     NOT NULL DEFAULT FALSE,
			client_reference     TEXT        NOT NULL DEFAULT '',
			event_at             TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX subscriptions_client_reference_idx
		   ON subscriptions (client_reference)`,
		`CREATE TABLE payment_events (
			id         BIGSERIAL PRIMARY KEY,
			event_id   TEXT        NOT NULL UNIQUE,
			event_type TEXT        NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)`,
{{- else}}
		`CREATE TABLE subscriptions (
			id                   INTEGER PRIMARY KEY AUTOINCREMENT,
			subscription_id      TEXT    NOT NULL UNIQUE,
			customer_id          TEXT    NOT NULL,
			status               TEXT    NOT NULL,
			price_id             TEXT    NOT NULL DEFAULT '',
			current_period_end   DATETIME,
			cancel_at_period_end BOOLEAN NOT NULL DEFAULT 0,
			client_reference     TEXT    NOT NULL DEFAULT '',
			event_at             DATETIME NOT NULL
		)`,
		`CREATE INDEX subscriptions_client_reference_idx
		   ON subscriptions (client_reference)`,
		`CREATE TABLE payment_events (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id   TEXT     NOT NULL UNIQUE,
			event_type TEXT     NOT NULL,
			created_at DATETIME NOT NULL
		)`,
{{- end}}
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("creating the payment tables: %w", err)
		}
	}
	return nil
}

func mig_20261111000000_create_subscriptions_table_down(tx *sql.Tx) error {
	for _, statement := range []string{
		`DROP TABLE IF EXISTS payment_events`,
		`DROP TABLE IF EXISTS subscriptions`,
	} {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}
