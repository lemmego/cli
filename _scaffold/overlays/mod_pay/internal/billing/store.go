// Package billing is this application's record of who has paid.
//
// It implements pay.Store, which is the half of a payments integration the
// module cannot write for you: the module knows what Stripe said, and this
// knows what this application believes afterwards. The expensive mistakes in
// billing are all on this side.
package billing

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lemmego/api/db"
	"github.com/lemmego/pay"
)

// Store writes subscription state and remembers which events it has seen.
type Store struct {
	pool    *sql.DB
	dialect db.Dialect
}

func New(conn db.Connection) *Store {
	return &Store{pool: conn.SQLDB(), dialect: conn.Dialect()}
}

var _ pay.Store = (*Store)(nil)

// Seen records that an event arrived and reports whether this is the first
// time.
//
// **Stripe delivers twice.** It retries until a delivery is acknowledged,
// and it retries on its own schedule even after a success it did not hear
// about — so the second delivery must do nothing. The whole implementation
// is a unique index on the event id and nothing cleverer: the insert either
// succeeds, which means first, or it conflicts, which means seen.
//
// Doing it as one statement rather than a SELECT and then an INSERT is the
// difference between a guard and a race. Two concurrent deliveries of one
// event both read "not seen" and both apply it.
func (s *Store) Seen(ctx pay.Context, event *pay.Event) (bool, error) {
	result, err := s.pool.ExecContext(ctx,
		s.rebind(`INSERT INTO payment_events (event_id, event_type, created_at)
		          VALUES (?, ?, ?)
		          ON CONFLICT (event_id) DO NOTHING`),
		event.ID, event.Type, event.CreatedAt)
	if err != nil {
		return false, fmt.Errorf("recording event %s: %w", event.ID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("recording event %s: %w", event.ID, err)
	}
	return affected == 1, nil
}

// Apply writes the event's subscription state.
//
// **It must be safe to call out of order.** Stripe makes no ordering promise
// and delivers concurrently, so the newer event has to win however they
// arrive. That is done as a condition on the write — the WHERE on the
// DO UPDATE — and not as a read followed by a write, because a read-then-write
// guard has a window exactly wide enough to lose the newer event. The
// symptom is a cancelled subscription that stays active, or an active one
// that stays cancelled, and nothing in any log says why.
func (s *Store) Apply(ctx pay.Context, event *pay.Event) error {
	sub := event.Subscription
	if sub == nil {
		// Not every event carries one — a completed checkout may only name
		// the customer. Nothing to write is not a failure.
		return nil
	}

	var periodEnd any
	if sub.CurrentPeriodEnd != nil {
		periodEnd = *sub.CurrentPeriodEnd
	}

	_, err := s.pool.ExecContext(ctx,
		s.rebind(`INSERT INTO subscriptions
		    (subscription_id, customer_id, status, price_id, current_period_end,
		     cancel_at_period_end, client_reference, event_at)
		  VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		  ON CONFLICT (subscription_id) DO UPDATE SET
		    customer_id          = excluded.customer_id,
		    status               = excluded.status,
		    price_id             = excluded.price_id,
		    current_period_end   = excluded.current_period_end,
		    cancel_at_period_end = excluded.cancel_at_period_end,
		    client_reference     = excluded.client_reference,
		    event_at             = excluded.event_at
		  WHERE excluded.event_at >= subscriptions.event_at`),
		sub.ID, sub.CustomerID, string(sub.Status), sub.PriceID, periodEnd,
		sub.CancelAtPeriodEnd, event.ClientReference, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("applying subscription %s: %w", sub.ID, err)
	}
	return nil
}

// Active reports the current subscription for one of this application's own
// users, by whatever reference was put in the checkout.
//
// **Entitlement is read from here and never from the redirect.** A visitor
// who lands on the success URL has not necessarily paid — they may have
// opened it directly — and the webhook is the only thing that knows.
func (s *Store) Active(ctx pay.Context, reference string) (*Record, error) {
	row := s.pool.QueryRowContext(ctx,
		s.rebind(`SELECT subscription_id, customer_id, status, price_id,
		                 current_period_end, cancel_at_period_end
		            FROM subscriptions
		           WHERE client_reference = ?
		        ORDER BY event_at DESC
		           LIMIT 1`), reference)

	var rec Record
	var periodEnd sql.NullTime
	if err := row.Scan(&rec.SubscriptionID, &rec.CustomerID, &rec.Status,
		&rec.PriceID, &periodEnd, &rec.CancelAtPeriodEnd); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if periodEnd.Valid {
		rec.CurrentPeriodEnd = &periodEnd.Time
	}
	return &rec, nil
}

// rebind turns ? placeholders into $N for Postgres.
//
// Written once here rather than branching in every statement above, because
// a statement maintained in two dialects is a statement maintained twice.
func (s *Store) rebind(query string) string {
	if s.dialect != db.Postgres {
		return query
	}

	out := make([]byte, 0, len(query)+8)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] != '?' {
			out = append(out, query[i])
			continue
		}
		n++
		out = append(out, []byte(fmt.Sprintf("$%d", n))...)
	}
	return string(out)
}
