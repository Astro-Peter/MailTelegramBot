package ImapAdapter

import (
	"MailTelegramBot/Models"
	"fmt"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"log"
	"mime"
	"net/mail"
	"strconv"
	"time"
)

type ImapAdapter struct {
	imapServer     string
	userName       string
	password       string
	imapClient     *imapclient.Client
	messageChannel chan Models.Email
	updatesChannel <-chan Models.MessageUpdate
	timeChannel    *time.Ticker
	emailMap       *Models.EmailHashMap
	mailAddresses  []string
	lastSearchTime time.Time
}

const (
	dayHours           = 24
	refreshRateSeconds = 20
)

func (adapter *ImapAdapter) AddAddress(address string) {
	adapter.mailAddresses = append(adapter.mailAddresses, address)
}

func (adapter *ImapAdapter) SetupChans(updates <-chan Models.MessageUpdate, emails chan Models.Email) {
	adapter.emailMap = Models.NewEmailHashMap()
	adapter.messageChannel = emails
	adapter.updatesChannel = updates
	adapter.timeChannel = time.NewTicker(time.Second * refreshRateSeconds)
	adapter.lastSearchTime = time.Now()
}

func (adapter *ImapAdapter) Setup(scanner Models.PrinterScanner) {
	err := adapter.setupImapTls(scanner)
	for err != nil {
		scanner.DisplayMessage(err.Error())
		err = adapter.setupImapTls(scanner)
	}

	err = adapter.login(scanner)
	for err != nil {
		scanner.DisplayMessage(err.Error())
		err = adapter.login(scanner)
	}
}

func (adapter *ImapAdapter) login(scanner Models.PrinterScanner) error {
	adapter.userName = scanner.GetAnswer("Username: ")
	adapter.password = scanner.GetAnswer("Password: ")
	loginResult := adapter.imapClient.Login(adapter.userName, adapter.password)
	if err := loginResult.Wait(); err != nil {
		return err
	}
	_, err := adapter.imapClient.Select("inbox", nil).Wait()
	if err != nil {
		return err
	}
	return err
}

func (adapter *ImapAdapter) reLogin() error {
	decode := &mime.WordDecoder{CharsetReader: charset.Reader}
	options := &imapclient.Options{
		WordDecoder: decode,
	}

	c, err := imapclient.DialTLS(fmt.Sprintf("%s:993", adapter.imapServer), options)
	if err != nil {
		return err
	}

	adapter.imapClient = c
	loginResult := adapter.imapClient.Login(adapter.userName, adapter.password)
	if err := loginResult.Wait(); err != nil {
		return err
	}
	_, err = adapter.imapClient.Select("inbox", nil).Wait()
	return err
}

func (adapter *ImapAdapter) setupImapTls(scanner Models.PrinterScanner) error {
	decode := &mime.WordDecoder{CharsetReader: charset.Reader}
	options := &imapclient.Options{
		WordDecoder: decode,
	}
	adapter.imapServer = scanner.GetAnswer("Give IMAP server address")

	c, err := imapclient.DialTLS(fmt.Sprintf("%s:993", adapter.imapServer), options)
	if err != nil {
		return err
	}
	adapter.imapClient = c
	return nil
}

func (adapter *ImapAdapter) Loop() {
	cnt := 0
	cntBoundary := int((time.Hour * dayHours) / (refreshRateSeconds * time.Second))
	for {
		select {
		case <-adapter.timeChannel.C:
			cnt++
			if cnt == cntBoundary {
				cnt = 0
				adapter.emailMap.Clear()
			}
			err := adapter.reLogin()
			if err != nil {
				log.Println(err)
				continue
			}
			uids := adapter.Search()
			if uids == nil {
				continue
			}
			adapter.Send(uids)
		case update := <-adapter.updatesChannel:
			err := adapter.reLogin()
			if err != nil {
				log.Println(err)
				continue
			}
			adapter.UpdateEmailStatus(update)
		}
	}
}

func (adapter *ImapAdapter) Search() imap.NumSet {
	start := time.Now()
	data, err := adapter.imapClient.UIDSearch(&imap.SearchCriteria{
		Since:  adapter.lastSearchTime,
		Before: start.Add(time.Hour * dayHours),
	}, nil).Wait()
	adapter.lastSearchTime = start
	if err != nil {
		log.Printf("imap error, %s", err.Error())
		return nil
	}
	return data.All
}

func (adapter *ImapAdapter) SearchTest() imap.NumSet {
	start := time.Now().Add(-time.Hour * dayHours)
	data, err := adapter.imapClient.UIDSearch(&imap.SearchCriteria{
		Since:  start,
		Before: time.Now().Add(time.Hour * dayHours),
	}, nil).Wait()
	adapter.lastSearchTime = time.Now()
	if err != nil {
		log.Printf("imap error, %s", err.Error())
		return nil
	}
	return data.All
}

func (adapter *ImapAdapter) Send(uids imap.NumSet) {
	options := &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{}},
	}
	fetchCmd := adapter.imapClient.Fetch(uids, options)
	defer fetchCmd.Close()

	for {
		send := true
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		var email Models.Email
		for {
			item := msg.Next()
			if item == nil {
				break
			}

			switch item := item.(type) {
			case imapclient.FetchItemDataUID:
				if adapter.emailMap.Check(item.UID) {
					send = false
					break
				}
				adapter.emailMap.Add(item.UID)
				email.Id = strconv.Itoa(int(item.UID))
			case imapclient.FetchItemDataBodySection:
				b, err := mail.ReadMessage(item.Literal)
				if err != nil {
					log.Printf("failed to read body section: %v", err)
				}
				dec := new(mime.WordDecoder)
				email.Sender, _ = dec.DecodeHeader(b.Header["From"][0])
				email.Receiver, _ = dec.DecodeHeader(b.Header["To"][0])
				email.Subject, _ = dec.DecodeHeader(b.Header["Subject"][0])
			}
		}
		if send {
			adapter.messageChannel <- email
		}
	}
}

func (adapter *ImapAdapter) UpdateEmailStatus(update Models.MessageUpdate) {
	parsed, _ := strconv.ParseInt(update.EmailId, 10, 32)
	seq := adapter.getSeqByUid(imap.UID(parsed))
	if seq == nil || len(seq) == 0 {
		return
	}
	var flags = []imap.Flag{imap.FlagFlagged}
	cmd := adapter.imapClient.Store(imap.SeqSetNum(uint32(seq[0])), &imap.StoreFlags{
		Op:     imap.StoreFlagsSet,
		Silent: false,
		Flags:  flags,
	}, nil)
	if err := cmd.Wait(); err != nil {
		log.Printf("imap error, %s", err.Error())
	}
}

func (adapter *ImapAdapter) getSeqByUid(uid imap.UID) []uint32 {
	var uids = []imap.UIDSet{imap.UIDSetNum(uid)}
	var d = &imap.SearchCriteria{
		UID: uids,
	}
	result, err := adapter.imapClient.Search(d, nil).Wait()
	if err != nil {
		log.Printf("search error: %s", err.Error())
		return nil
	}
	return result.AllSeqNums()
}
