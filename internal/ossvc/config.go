package ossvc

import (
	"fmt"
	"time"

	"github.com/kardianos/service"
)

// configureServiceOptions defines additional settings of the service
// configuration.  conf must not be nil.
//
//lint:ignore U1000 TODO(e.burkov): Use.
func configureServiceOptions(conf *service.Config, versionInfo string) {
	conf.Option["SvcInfo"] = fmt.Sprintf("%s %s", versionInfo, time.Now())

	configureOSOptions(conf)
}
