package Models

import (
	"fmt"
	"log"
)

type Email struct {
	Id       string
	Sender   string
	Receiver string
	Subject  string
}

func (e *Email) String() string {
	log.Println(e.Subject, e.Receiver, e.Sender, e.Id)
	return fmt.Sprintf("From: %s\nTo: %s\n\n%s", e.Sender, e.Receiver, e.Subject)
}
