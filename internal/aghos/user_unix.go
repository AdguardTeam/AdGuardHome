//go:build darwin || freebsd || linux || openbsd

package aghos

import (
	"fmt"
	"os/user"
	"strconv"
	"syscall"
)

func setGroup(groupName string) (err error) {
	g, err := user.LookupGroup(groupName)
	if err != nil {
		return fmt.Errorf("looking up group: %w", err)
	}

	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return fmt.Errorf("parsing gid: %w", err)
	}

	err = syscall.Setgid(gid)
	if err != nil {
		return fmt.Errorf("setting gid: %w", err)
	}

	return nil
}

func setUser(userName string) (err error) {
	u, err := user.Lookup(userName)
	if err != nil {
		return fmt.Errorf("looking up user: %w", err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("parsing uid: %w", err)
	}

	err = syscall.Setuid(uid)
	if err != nil {
		return fmt.Errorf("setting uid: %w", err)
	}

	return nil
}
