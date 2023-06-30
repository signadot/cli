package local

import (
	"errors"
	"time"

	"github.com/signadot/libconnect/common/pmlock"
)

func Lock(dir string) error {
	l := pmlock.NewLocker(dir)
	return pmlock.TryLock(l, "local-connect.lock", 3*time.Second)
}

func IsLocked(dir string) (bool, error) {
	l := pmlock.NewLocker(dir)
	err := pmlock.TryLock(l, "local-connect.lock", 100*time.Millisecond)
	if err == nil {
		err = l.Unlock("local-connect.lock")
		if err != nil {
			return false, err
		}
		return false, nil
	}
	if errors.Is(err, pmlock.ErrLocked) {
		return true, nil
	}
	return false, err
}

func Unlock(dir string) error {
	l := pmlock.NewLocker(dir)
	return l.Unlock("local-connect.lock")
}
