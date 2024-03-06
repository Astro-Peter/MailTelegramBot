package Models

import "github.com/emersion/go-imap/v2"

type NewStatus bool

type MessageUpdate struct {
	EmailId   string
	SetStatus imap.StoreFlagsOp
}
