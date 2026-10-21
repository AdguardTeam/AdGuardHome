// Package rulelist contains the implementation of the standard rule-list
// filter that wraps an urlfilter filtering-engine.
//
// TODO(a.garipov): Add a new update worker.
package rulelist

import (
	"fmt"
	"math"
	"uuid"

	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/c2h5oh/datasize"
)

// DefaultRuleBufSize is the default length of a buffer used to read a line with
// a filtering rule, in bytes.
//
// TODO(a.garipov): Consider using [datasize.ByteSize].  It is currently only
// used as an int.
const DefaultRuleBufSize = 1024

// DefaultMaxRuleListSize is the default maximum filtering-rule list size.
const DefaultMaxRuleListSize = 256 * datasize.MB

// APIID is the type for the rule-list IDs used in the HTTP API.
type APIID int64

// The IDs of built-in filter lists for the HTTP API.
//
// NOTE:  Do not change without the need for it and keep in sync with
// client/src/helpers/constants.ts.
const (
	APIIDCustom          APIID = 0
	APIIDEtcHosts        APIID = -1
	APIIDBlockedService  APIID = -2
	APIIDParentalControl APIID = -3
	APIIDSafeBrowsing    APIID = -4
	APIIDSafeSearch      APIID = -5
)

// The IDs of built-in filter lists.  The IDs for the blocked-service and the
// safe-search filters are chosen so that they equal to their [APIID]
// counterparts when converted to it.
//
// NOTE:  Keep in sync with [APIIDCustom] etc.
//
// TODO(d.kolyshev): Add URLFilterIDLegacyRewrite here and to the UI.
const (
	IDCustom         rules.ListID = rules.ListID(APIIDCustom)
	IDBlockedService rules.ListID = math.MaxUint64 - rules.ListID(-APIIDBlockedService) + 1
	IDSafeSearch     rules.ListID = math.MaxUint64 - rules.ListID(-APIIDSafeSearch) + 1
)

// UID is the type for the unique IDs of filtering-rule lists.
type UID uuid.UUID

// NewUID returns a new filtering-rule list UID.
func NewUID() (uid UID) {
	return UID(uuid.NewV7())
}

// type check
var _ fmt.Stringer = UID{}

// String implements the [fmt.Stringer] interface for UID.
func (id UID) String() (s string) {
	return uuid.UUID(id).String()
}

// Common engine names.
const (
	EngineNameAllow  = "allow"
	EngineNameBlock  = "block"
	EngineNameCustom = "custom"
)
