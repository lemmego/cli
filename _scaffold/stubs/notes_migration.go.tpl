package migrations

import (
	"database/sql"
	"fmt"

	"github.com/lemmego/migration"
)

func init() {
	migration.GetMigrator().AddMigration(&migration.Migration{
		Version: "20261112000000",
		Up:      mig_20261112000000_create_notes_table_up,
		Down:    mig_20261112000000_create_notes_table_down,
	})
}

// The notes table, as something for the search example to search.
//
// Raw SQL rather than the schema builder, because neither of the two things
// that make this searchable can be expressed by it: a generated tsvector
// column and a GIN index. Retrofitting them later means a backfill on a live
// table, so they are cheap now and a rewrite afterwards.
//
// **The vector is a generated column, not a trigger and not application
// code.** The database keeps it in step with the row, so it is correct the
// moment the row is written — including for a row written by a migration, by
// psql, or by a future service nobody has written yet. That is also why this
// module needs nothing synced: there is no second copy of the data to fall
// behind.
//
// setweight is what makes a title match outrank a body match. Without it
// every column counts the same and a note that merely mentions a word ranks
// alongside one that is about it.
func mig_20261112000000_create_notes_table_up(tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE notes (
			id         BIGSERIAL PRIMARY KEY,
			title      TEXT        NOT NULL,
			author     TEXT        NOT NULL DEFAULT '',
			body       TEXT        NOT NULL DEFAULT '',
			published  BOOLEAN     NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		// 'english' has to match the Table's Config, or the query is parsed
		// with one dictionary and the document with another — which finds
		// nothing for any word either one stems.
		`ALTER TABLE notes ADD COLUMN search_vector tsvector
		   GENERATED ALWAYS AS (
		     setweight(to_tsvector('english', coalesce(title, '')),  'A') ||
		     setweight(to_tsvector('english', coalesce(author, '')), 'B') ||
		     setweight(to_tsvector('english', coalesce(body, '')),   'C')
		   ) STORED`,

		`CREATE INDEX notes_search_vector_idx ON notes USING GIN (search_vector)`,

		// Partial, because every read goes through the visibility rule and
		// the index's whole size is then the rows that can be returned.
		`CREATE INDEX notes_published_idx ON notes (id) WHERE published = true`,
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("creating the notes table: %w", err)
		}
	}
	return nil
}

func mig_20261112000000_create_notes_table_down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS notes`)
	return err
}
