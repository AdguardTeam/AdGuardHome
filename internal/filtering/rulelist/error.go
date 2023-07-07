package rulelist

import "github.com/AdguardTeam/golibs/errors"

// ErrHTML is returned by [Parser.Parse] if the data is likely to be HTML.
//
// TODO(a.garipov): This error is currently returned to the UI.  Stop that and
// make it all-lowercase.
const ErrHTML errors.Error = "data is HTML, not plain text"
