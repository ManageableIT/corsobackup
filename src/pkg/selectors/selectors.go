package selectors

import (
	"strconv"

	"github.com/pkg/errors"
)

type service int

//go:generate stringer -type=service -linecomment
const (
	ServiceUnknown  service = iota // Unknown Service
	ServiceExchange                // Exchange
)

var ErrorBadSelectorCast = errors.New("wrong selector service type")

const (
	scopeKeyGranularity = "granularity"
	scopeKeyCategory    = "category"
)

const (
	// All is the wildcard value used to express "all data of <type>"
	// Ex: Events(u1, All) => all events for user u1.
	All = "*"
)

// The core selector.  Has no api for setting or retrieving data.
// Is only used to pass along more specific selector instances.
type Selector struct {
	TenantID string              // The tenant making the request.
	service  service             // The service scope of the data.  Exchange, Teams, Sharepoint, etc.
	scopes   []map[string]string // A slice of scopes.  Expected to get cast to fooScope within each service handler.
}

// helper for specific selector instance constructors.
func newSelector(tenantID string, s service) Selector {
	return Selector{
		TenantID: tenantID,
		service:  s,
		scopes:   []map[string]string{},
	}
}

// Service return the service enum for the selector.
func (s Selector) Service() service {
	return s.service
}

func badCastErr(cast, is service) error {
	return errors.Wrapf(ErrorBadSelectorCast, "%s service is not %s", cast, is)
}

type scopeGranularity int

// granularity expresses the breadth of the request
const (
	GranularityUnknown scopeGranularity = iota
	SingleItem
	AllIn
)

// String complies with the stringer interface, so that granularities
// can be added into the scope map.
func (g scopeGranularity) String() string {
	return strconv.Itoa(int(g))
}

func granularityOf(selector map[string]string) scopeGranularity {
	return scopeGranularity(getIota(selector, scopeKeyGranularity))
}

// retrieves the iota, stored as a string, and transforms it to
// an int.  Any errors will return a 0 by default.
func getIota(m map[string]string, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return i
}