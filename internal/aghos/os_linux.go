//go:build linux

package aghos

import (
	"io"
	"os"
	"syscall"

	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/stringutil"
)

func setRlimit(val uint64) (err error) {
	var rlim syscall.Rlimit
	rlim.Max = val
	rlim.Cur = val

	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
}

func haveAdminRights() (bool, error) {
	// The error is nil because the platform-independent function signature
	// requires returning an error.
	return os.Getuid() == 0, nil
}

func isOpenWrt() (ok bool) {
	const etcReleasePattern = "etc/*release*"

	var err error
	ok, err = FileWalker(func(r io.Reader) (_ []string, cont bool, err error) {
		const osNameData = "openwrt"

		// This use of ReadAll is now safe, because FileWalker's Walk()
		// have limited r.
		var data []byte
		data, err = io.ReadAll(r)
		if err != nil {
			return nil, false, err
		}

		return nil, !stringutil.ContainsFold(string(data), osNameData), nil
	}).Walk(osutil.RootDirFS(), etcReleasePattern)

	return err == nil && ok
}
