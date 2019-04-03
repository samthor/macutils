package macutils

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	// ErrNoSyscallStat is an internal error that a syscall is not available.
	ErrNoSyscallStat = errors.New("no syscall.Stat_t available")

	// ErrInvalidUser indicates that the user could not be determined.
	ErrInvalidUser = errors.New("couldn't get username")
)

const (
	consolePath = "/dev/console"
)

// Login describes a logged-in user on macOS, and when they logged in.
type Login struct {
	User user.User
	When time.Time
}

// CurrentLogin returns the current logged-in user on macOS.
func CurrentLogin() (lu Login, err error) {
	s, err := os.Stat(consolePath)
	if err != nil {
		return lu, err
	}
	lu.When = s.ModTime()

	ss, ok := s.Sys().(*syscall.Stat_t)
	if !ok {
		return lu, ErrNoSyscallStat
	}

	suid := fmt.Sprintf("%d", ss.Uid) // nb. don't use string()
	uo, _ := user.LookupId(suid)
	if uo != nil {
		lu.User = *uo
	} else {
		// fallback to stat as LookupId uses /etc/passwd, not populated on macOS
		out, err := exec.Command("/usr/bin/stat", "-f", `%Su`, consolePath).Output()
		if err != nil {
			return lu, err
		}
		if len(out) > 0 && out[0] == '(' {
			// nb. this happens while inside detached screen
			log.Printf("can't find username via stat, got uid: %s", string(out))
			log.Printf("...(on macOS, try leaving `screen`)")
			out = out[:0]
		}
		lu.User = user.User{Uid: suid, Username: string(out)}
	}

	if lu.User.Username == "" {
		return lu, ErrInvalidUser
	}
	return lu, nil
}

// LoginWatcher watches for login change events.
type LoginWatcher struct {
	Change chan Login // listen for logins and changems
	Errors chan error // listen for errors

	shutdownCh chan interface{}
	prev       user.User
}

// Close stops watching for login changes.
func (lw *LoginWatcher) Close() {
	close(lw.shutdownCh)
}

func (lw *LoginWatcher) update() {
	lu, err := CurrentLogin()
	if err != nil {
		lw.Errors <- err
		return
	}
	if lu.User.Username == lw.prev.Username {
		return
	}
	lw.prev = lu.User

	lw.Change <- lu
}

// SubscribeLogin subscribes to login change events and returns a LoginWatcher.
func SubscribeLogin() (*LoginWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = watcher.Add(consolePath)
	if err != nil {
		return nil, err
	}

	lw := &LoginWatcher{
		Change:     make(chan Login),
		Errors:     make(chan error),
		shutdownCh: make(chan interface{}),
	}

	go func() {
		defer watcher.Close()
		lw.update()

		for {
			select {
			case <-lw.shutdownCh:
				return

			case <-watcher.Events:
				// TODO(samthor): filter to changes
				lw.update()

			case err := <-watcher.Errors:
				lw.Errors <- err
			}
		}

	}()

	return lw, nil
}
