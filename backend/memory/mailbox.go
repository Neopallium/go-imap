package memory

import (
	"io/ioutil"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/backend/backendutil"
)

var Delimiter = "/"

type Mailbox struct {
	sync.RWMutex

	Attributes  []string
	Subscribed  bool
	Messages    []*Message
	UidValidity uint32

	name string
	user *User
}

func NewMailbox(user *User, name string, specialUse string) *Mailbox {
	mbox := &Mailbox{
		name: name, user: user,
		UidValidity: 1, // Use 1 for tests.  Should use timestamp instead.
		Messages:    []*Message{},
	}
	if specialUse != "" {
		mbox.Attributes = []string{specialUse}
	}
	return mbox
}

func (mbox *Mailbox) Name() string {
	return mbox.name
}

func (mbox *Mailbox) Info() (*imap.MailboxInfo, error) {
	mbox.RLock()
	defer mbox.RUnlock()

	info := &imap.MailboxInfo{
		Attributes: mbox.Attributes,
		Delimiter:  Delimiter,
		Name:       mbox.name,
	}
	return info, nil
}

func (mbox *Mailbox) uidNext() uint32 {
	var uid uint32
	for _, msg := range mbox.Messages {
		if msg.Uid > uid {
			uid = msg.Uid
		}
	}
	uid++
	return uid
}

func (mbox *Mailbox) unseenSeqNum() uint32 {
	for i, msg := range mbox.Messages {
		seqNum := uint32(i + 1)

		seen := false
		for _, flag := range msg.Flags {
			if flag == imap.SeenFlag {
				seen = true
				break
			}
		}

		if !seen {
			return seqNum
		}
	}
	return 0
}

func (mbox *Mailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	mbox.RLock()
	defer mbox.RUnlock()

	status := imap.NewMailboxStatus(mbox.name, items)
	status.Flags = []string{
		imap.AnsweredFlag, imap.FlaggedFlag, imap.DeletedFlag, imap.SeenFlag, imap.DraftFlag, "nonjunk",
	}
	status.PermanentFlags = []string{
		imap.AnsweredFlag, imap.FlaggedFlag, imap.DeletedFlag, imap.SeenFlag, imap.DraftFlag, "nonjunk", "\\*",
	}
	status.UnseenSeqNum = mbox.unseenSeqNum()

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = uint32(len(mbox.Messages))
		case imap.StatusUidNext:
			status.UidNext = mbox.uidNext()
		case imap.StatusUidValidity:
			status.UidValidity = mbox.UidValidity
		case imap.StatusRecent:
			status.Recent = 0 // TODO
		case imap.StatusUnseen:
			status.Unseen = 0 // TODO
		}
	}

	return status, nil
}

func (mbox *Mailbox) SetSubscribed(subscribed bool) error {
	mbox.Lock()
	defer mbox.Unlock()

	mbox.Subscribed = subscribed

	return nil
}

func (mbox *Mailbox) Check() error {
	return nil
}

func (mbox *Mailbox) ListMessages(uid bool, seqSet *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error {
	mbox.RLock()
	defer mbox.RUnlock()
	defer close(ch)

	for i, msg := range mbox.Messages {
		seqNum := uint32(i + 1)

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = seqNum
		}
		if !seqSet.Contains(id) {
			continue
		}

		m, err := msg.Fetch(seqNum, items)
		if err != nil {
			continue
		}

		ch <- m
	}

	return nil
}

func (mbox *Mailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	mbox.RLock()
	defer mbox.RUnlock()

	var ids []uint32
	for i, msg := range mbox.Messages {
		seqNum := uint32(i + 1)

		ok, err := msg.Match(seqNum, criteria)
		if err != nil || !ok {
			continue
		}

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = seqNum
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (mbox *Mailbox) CreateMessage(flags []string, date time.Time, body imap.Literal) error {
	mbox.Lock()
	defer mbox.Unlock()

	if date.IsZero() {
		date = time.Now()
	}

	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	mbox.Messages = append(mbox.Messages, &Message{
		Uid:   mbox.uidNext(),
		Date:  date,
		Size:  uint32(len(b)),
		Flags: flags,
		Body:  b,
	})
	return nil
}

func (mbox *Mailbox) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, op imap.FlagsOp, flags []string) error {
	mbox.Lock()
	defer mbox.Unlock()

	for i, msg := range mbox.Messages {
		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		msg.Flags = backendutil.UpdateFlags(msg.Flags, op, flags)
	}

	return nil
}

func (mbox *Mailbox) CopyMessages(uid bool, seqset *imap.SeqSet, destName string) error {
	mbox.Lock()
	defer mbox.Unlock()

	dest, ok := mbox.user.mailboxes[destName]
	if !ok {
		return backend.ErrNoSuchMailbox
	}

	for i, msg := range mbox.Messages {
		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		msgCopy := *msg
		msgCopy.Uid = dest.uidNext()
		dest.Messages = append(dest.Messages, &msgCopy)
	}

	return nil
}

func (mbox *Mailbox) MoveMessages(uid bool, seqset *imap.SeqSet, destName string) error {
	mbox.Lock()
	defer mbox.Unlock()

	dest, ok := mbox.user.mailboxes[destName]
	if !ok {
		return backend.ErrNoSuchMailbox
	}

	for i, msg := range mbox.Messages {
		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		msgCopy := *msg
		msgCopy.Uid = dest.uidNext()
		dest.Messages = append(dest.Messages, &msgCopy)
		// Mark source message as deleted
		msg.Flags = backendutil.UpdateFlags(msg.Flags, imap.AddFlags, []string{imap.DeletedFlag})
	}

	return mbox.expunge()
}

func (mbox *Mailbox) expunge() error {
	for i := len(mbox.Messages) - 1; i >= 0; i-- {
		msg := mbox.Messages[i]

		deleted := false
		for _, flag := range msg.Flags {
			if flag == imap.DeletedFlag {
				deleted = true
				break
			}
		}

		if deleted {
			mbox.Messages = append(mbox.Messages[:i], mbox.Messages[i+1:]...)
		}
	}

	return nil
}

func (mbox *Mailbox) Expunge() error {
	mbox.Lock()
	defer mbox.Unlock()

	return mbox.expunge()
}
