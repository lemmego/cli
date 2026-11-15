// Package mailables holds this application's emails, one type per email.
//
// A Mailable is a plain value: it carries its own data, renders its own body
// and names its own subject, so it can be constructed and asserted on in a
// test without an application, a transport or a relay. That is why Build
// takes the renderer as an argument rather than reaching for one.
package mailables

import (
	"fmt"

	"github.com/lemmego/mail"
)

// Welcome is the worked example.
type Welcome struct {
	Name  string
	Email string
	// URL is where the message sends the reader. Built by the caller, which
	// is the only place that knows this application's base URL.
	URL string
}

// Build renders the body and names the subject.
//
// Both parts are rendered: welcome.gohtml for the HTML and, when it exists,
// welcome.gotxt for the plain-text alternative. Sending both is not
// politeness — a message with no text part scores worse with every spam
// filter, and some clients show the HTML source when there is nothing else.
func (w Welcome) Build(ctx mail.Context, r mail.Renderer) (*mail.Message, error) {
	html, text, err := r.Render("welcome", w)
	if err != nil {
		return nil, fmt.Errorf("rendering the welcome email: %w", err)
	}

	return &mail.Message{
		To:      []mail.Address{{Name: w.Name, Email: w.Email}},
		Subject: fmt.Sprintf("Welcome, %s", w.Name),
		HTML:    html,
		Text:    text,
	}, nil
}

// The cheapest guard there is, and worth keeping on every Mailable: one line,
// checked at compile time, and it catches a Build written with the wrong
// signature — which otherwise fails at the point of sending, to a real
// person.
var _ mail.Mailable = Welcome{}
