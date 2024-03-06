package Models

import (
	"github.com/emersion/go-imap/v2"
	"log"
	"time"
)

type EmailHashMap struct {
	emails map[imap.UID]time.Time
}

func NewEmailHashMap() *EmailHashMap {
	emails := make(map[imap.UID]time.Time)
	return &EmailHashMap{emails: emails}
}

const (
	dayHours = 24
)

func (e *EmailHashMap) Check(uid imap.UID) bool {
	_, ok := e.emails[uid]
	return ok
}

func (e *EmailHashMap) Add(uid imap.UID) {
	if _, ok := e.emails[uid]; ok {
		return
	}
	e.emails[uid] = time.Now()
}

func (e *EmailHashMap) Clear() {
	log.Println("clear")
	for uid, t := range e.emails {
		if t.Before(time.Now().Add(-dayHours * time.Hour)) {
			delete(e.emails, uid)
		}
	}
}
