package memory

import (
	"errors"
	"strconv"
	"sync"
	"time"

	specialuse "github.com/emersion/go-imap-specialuse"
	"github.com/emersion/go-imap/backend"
)

type User struct {
	sync.RWMutex

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

	t := time.Now()
	body := "From: contact@example.org\r\n" +
		"To: " + username + "@example.org\r\n" +
		"Subject: A little message, just for you\r\n" +
		"Date: " + t.Format("Mon, 2 Jan 2006 15:04:05 -0700") + "\r\n" +
		"Message-ID: <" + strconv.Itoa(t.Nanosecond()) + "@localhost/>\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"Hi " + username + " there :)"

	user.createMailbox("INBOX", "")
	inbox := user.mailboxes["INBOX"]
	inbox.Messages = []*Message{
		{
			Uid:   1,
			Date:  t,
			Flags: []string{},
			Size:  uint32(len(body)),
			Body:  []byte(body),
		},
	}
	user.createMailbox("Sent", specialuse.Sent)
	user.createMailbox("Drafts", specialuse.Drafts)
	user.createMailbox("Queue", "")
	user.createMailbox("Trash", specialuse.Trash)

	return user
}

func (u *User) Username() string {
	return u.username
}

func (u *User) ListMailboxes(subscribed bool) (mailboxes []backend.Mailbox, err error) {
	u.RLock()
	defer u.RUnlock()

	for _, mailbox := range u.mailboxes {
		if subscribed && !mailbox.Subscribed {
			continue
		}

		mailboxes = append(mailboxes, mailbox)
	}
	return
}

func (u *User) GetMailbox(name string) (mailbox backend.Mailbox, err error) {
	u.RLock()
	defer u.RUnlock()

	mailbox, ok := u.mailboxes[name]
	if !ok {
		err = errors.New("No such mailbox")
	}
	return
}

func (u *User) createMailbox(name string, special_use string) error {
	if _, ok := u.mailboxes[name]; ok {
		return errors.New("Mailbox already exists")
	}

	u.mailboxes[name] = NewMailbox(u, name, special_use)
	return nil
}

func (u *User) CreateMailbox(name string) error {
	u.Lock()
	defer u.Unlock()

	return u.createMailbox(name, "")
}

func (u *User) DeleteMailbox(name string) error {
	u.Lock()
	defer u.Unlock()

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
	u.Lock()
	defer u.Unlock()

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
