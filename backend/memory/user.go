package memory

import (
	"errors"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
)

type User struct {
	backend   *Backend
	username  string
	password  string
	mailboxes map[string]*Mailbox
}

func NewUser(backend *Backend, username string, password string) *User {
	user := &User{
		backend:   backend,
		username:  username,
		password:  password,
		mailboxes: map[string]*Mailbox{},
	}

	// Message for tests
	body := "From: contact@example.org\r\n" +
		"To: contact@example.org\r\n" +
		"Subject: A little message, just for you\r\n" +
		"Date: Wed, 11 May 2016 14:31:59 +0000\r\n" +
		"Message-ID: <0000000@localhost/>\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"Hi there :)"

	user.CreateMailbox("INBOX")
	inbox := user.mailboxes["INBOX"]
	inbox.Messages = []*Message{
		{
			Uid:   6,
			Date:  time.Now(),
			Flags: []string{imap.SeenFlag},
			Size:  uint32(len(body)),
			Body:  []byte(body),
		},
	}
	user.CreateMailbox("Sent")
	user.CreateMailbox("Drafts")
	user.CreateMailbox("Queue")
	user.CreateMailbox("Trash")

	return user
}

func (u *User) Username() string {
	return u.username
}

func (u *User) ListMailboxes(subscribed bool) (mailboxes []backend.Mailbox, err error) {
	for _, mailbox := range u.mailboxes {
		if subscribed && !mailbox.Subscribed {
			continue
		}

		mailboxes = append(mailboxes, mailbox)
	}
	return
}

func (u *User) GetMailbox(name string) (mailbox backend.Mailbox, err error) {
	mailbox, ok := u.mailboxes[name]
	if !ok {
		err = errors.New("No such mailbox")
	}
	return
}

func (u *User) CreateMailbox(name string) error {
	if _, ok := u.mailboxes[name]; ok {
		return errors.New("Mailbox already exists")
	}

	u.mailboxes[name] = NewMailbox(u, name)
	return nil
}

func (u *User) DeleteMailbox(name string) error {
	if name == "INBOX" {
		return errors.New("Cannot delete INBOX")
	}
	if _, ok := u.mailboxes[name]; !ok {
		return errors.New("No such mailbox")
	}

	delete(u.mailboxes, name)
	return nil
}

func (u *User) RenameMailbox(existingName, newName string) error {
	mbox, ok := u.mailboxes[existingName]
	if !ok {
		return errors.New("No such mailbox")
	}

	u.mailboxes[newName] = &Mailbox{
		name:     newName,
		Messages: mbox.Messages,
		user:     u,
	}

	mbox.Messages = nil

	if existingName != "INBOX" {
		delete(u.mailboxes, existingName)
	}

	return nil
}

func (u *User) Logout() error {
	return nil
}
