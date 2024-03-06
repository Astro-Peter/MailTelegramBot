package main

import (
	"MailTelegramBot/ImapAdapter"
	"MailTelegramBot/Models"
	"MailTelegramBot/TelegramBot"
)

func main() {
	var updates = make(chan Models.MessageUpdate, 128)
	var emails = make(chan Models.Email, 128)
	var scanner Models.StdioPrinterScanner
	var tgBot = &TelegramBot.MailBot{}
	tgBot.SetupChans(emails, updates)
	tgBot.Setup(scanner)

	var mailBot = &ImapAdapter.ImapAdapter{}
	mailBot.SetupChans(updates, emails)
	mailBot.Setup(scanner)

	go tgBot.Start()
	mailBot.Send(mailBot.SearchTest())
	go mailBot.Loop()
	select {}
}
