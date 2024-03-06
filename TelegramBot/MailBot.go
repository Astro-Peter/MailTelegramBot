package TelegramBot

import (
	"MailTelegramBot/Models"
	"fmt"
	"github.com/emersion/go-imap/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"strconv"
	"strings"
)

type MailBot struct {
	bot      *tgbotapi.BotAPI
	messages <-chan Models.Email
	chatId   int64
	updates  chan Models.MessageUpdate
}

// Scanner receives prompt for required information and returns the info
type Scanner func(string2 string) string

func GetIdAndStatus(button string) (string, string) {
	a := strings.Split(button, ",")
	return a[0], a[1]
}

func SetImportantKeyboard(emailId string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("set important", fmt.Sprintf("ok,%s", emailId)),
		),
	)
}

func SetNotImportantKeyboard(emailId string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("set not important", fmt.Sprintf("notok,%s", emailId)),
		),
	)
}

func (bot *MailBot) SetupChans(
	messages <-chan Models.Email,
	updates chan Models.MessageUpdate) {
	bot.messages = messages
	bot.updates = updates
}

// Setup function to set up bot using some Scanner
func (bot *MailBot) Setup(scanner Models.PrinterScanner) {
	err := bot.setupTgBot(scanner)
	for err != nil {
		scanner.DisplayMessage(err.Error())
		err = bot.setupTgBot(scanner)
	}

	err = bot.setupId(scanner)
	for err != nil {
		scanner.DisplayMessage(err.Error())
		err = bot.setupId(scanner)
	}
}

func (bot *MailBot) setupTgBot(scanner Models.PrinterScanner) error {
	botApiKey := scanner.GetAnswer("Provide bot api key")
	api, err := tgbotapi.NewBotAPI(botApiKey)
	if err != nil {
		return err
	}

	bot.bot = api
	return nil
}

func (bot *MailBot) setupId(scanner Models.PrinterScanner) error {
	userId := scanner.GetAnswer("Provide user id")
	chatId, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		return err
	}

	bot.chatId = chatId
	return nil
}

// TestSetup used for debugging
func (bot *MailBot) TestSetup(
	tgBot *tgbotapi.BotAPI,
	messages <-chan Models.Email,
	id int64) {
	bot.bot = tgBot
	bot.messages = messages
	bot.chatId = id
}

// Start to activate the bot. Should run in goroutine at the same
// time as the IMAP client
func (bot *MailBot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.bot.GetUpdatesChan(u)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case update := <-updates:
			bot.Handle(update)
		case email := <-bot.messages:
			bot.Send(email)
		}
	}
}

// Parse parses messages and does the required operation, will be implemented later
func (bot *MailBot) Parse(message *tgbotapi.Message) {
	switch message.Command() {
	case "add":
		bot.Add(message)
	}
}

// Handle handles updates. Should either Parse a message,
// or change email status, no other update is expected
func (bot *MailBot) Handle(update tgbotapi.Update) {
	if update.Message != nil {
		bot.Parse(update.Message)
	} else if update.CallbackQuery != nil {
		bot.UpdateMessageStatus(update.CallbackQuery)
		bot.ChangeInlineKeyboard(update.CallbackQuery)
	}
}

// Send sends a new email to user
func (bot *MailBot) Send(email Models.Email) {
	msg := tgbotapi.NewMessage(bot.chatId, email.String())
	msg.ReplyMarkup = SetImportantKeyboard(email.Id)
	_, err := bot.bot.Send(msg)
	if err != nil {
		log.Printf("tg error, %s", err.Error())
	}
}

// ChangeInlineKeyboard changes keyboard for message to the opposite one
// should only be called after user changes email status
func (bot *MailBot) ChangeInlineKeyboard(query *tgbotapi.CallbackQuery) error {
	var edit tgbotapi.EditMessageReplyMarkupConfig
	status, id := GetIdAndStatus(query.Data)
	if status == "ok" {
		keyboard := SetNotImportantKeyboard(id)
		edit = tgbotapi.NewEditMessageReplyMarkup(query.Message.Chat.ID, query.Message.MessageID, keyboard)
	} else {
		keyboard := SetImportantKeyboard(id)
		edit = tgbotapi.NewEditMessageReplyMarkup(query.Message.Chat.ID, query.Message.MessageID, keyboard)
	}
	_, err := bot.bot.Send(edit)
	return err
}

// UpdateMessageStatus changes email read status to the opposite on IMAP server
func (bot *MailBot) UpdateMessageStatus(query *tgbotapi.CallbackQuery) {
	var status imap.StoreFlagsOp
	if query.Data == "ok" {
		status = 0
	} else {
		status = 2
	}
	_, id := GetIdAndStatus(query.Data)
	bot.updates <- Models.MessageUpdate{
		EmailId:   id,
		SetStatus: status,
	}
}

func (bot *MailBot) Add(message *tgbotapi.Message) {

}
