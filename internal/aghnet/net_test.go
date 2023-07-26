package aghnet_test

import (
	"io/fs"
	"os"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

// testdata is the filesystem containing data for testing the package.
var testdata fs.FS = os.DirFS("./testdata")
