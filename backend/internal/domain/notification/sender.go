package notification

import "context"

// EmailSender abstracts email delivery so the domain doesn't depend on SMTP.
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}
