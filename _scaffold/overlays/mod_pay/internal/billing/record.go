package billing

import "time"

// Record is what this application believes about one subscription.
//
// Its own type rather than pay.Subscription, because this is the stored
// shape: it holds what was written by the last webhook that won, not what
// Stripe would answer if asked now. The two diverge for as long as a
// delivery is in flight, and conflating them is how a page shows a state no
// row has.
type Record struct {
	SubscriptionID    string     `json:"subscription_id"`
	CustomerID        string     `json:"customer_id"`
	Status            string     `json:"status"`
	PriceID           string     `json:"price_id"`
	CurrentPeriodEnd  *time.Time `json:"current_period_end"`
	CancelAtPeriodEnd bool       `json:"cancel_at_period_end"`
}

// Entitled reports whether this record should unlock paid features.
//
// Two days of grace past the period end, deliberately. A renewal webhook
// that is late or lost would otherwise lock somebody out of something they
// have paid for, and the failure is silent on our side and immediate on
// theirs. Grace is the cheaper mistake.
func (r *Record) Entitled(now time.Time) bool {
	if r == nil {
		return false
	}
	switch r.Status {
	case "active", "trialing":
	default:
		return false
	}
	if r.CurrentPeriodEnd == nil {
		return true
	}
	return now.Before(r.CurrentPeriodEnd.Add(48 * time.Hour))
}
