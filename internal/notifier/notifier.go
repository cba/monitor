package notifier

import "context"

// Notifier sends alert notifications.
type Notifier interface {
	// Name returns the notifier type identifier.
	Name() string

	// Send delivers a notification.
	Send(ctx context.Context, title, content string) error
}
