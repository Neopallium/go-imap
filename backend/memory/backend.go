// A memory backend.
package memory

import (
	"errors"
	"sync"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
)

type Backend struct {
	sync.RWMutex

	users map[string]*User

	updates chan backend.Update
}

func (be *Backend) Login(_ *imap.ConnInfo, username, password string) (backend.User, error) {
	be.Lock()
	defer be.Unlock()

	user, ok := be.users[username]
	// auto create users.
	if !ok {
		user = be.addUser(username, password)
	}
	if ok && user.password == password {
		return user, nil
	}

	return nil, errors.New("Bad username or password")
}

func (be *Backend) addUser(username, password string) *User {
	user := NewUser(be, username, password)
	be.users[username] = user
	return user
}

func (be *Backend) Updates() <-chan backend.Update {
	return be.updates
}

func (be *Backend) PushUpdate(update backend.Update) {
	be.updates <- update
}

func New() *Backend {
	return &Backend{
		users:   make(map[string]*User),
		updates: make(chan backend.Update),
	}
}
