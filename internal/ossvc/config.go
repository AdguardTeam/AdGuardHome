package ossvc

import (
	"fmt"
	"time"

	"github.com/kardianos/service"
)

// ConfigureServiceOptions defines additional settings of the service
// configuration.  conf must not be nil.
func ConfigureServiceOptions(conf *service.Config, now time.Time, versionInfo string) {
	if conf.Option == nil {
		conf.Option = map[string]any{}
	}

	conf.Option["SvcInfo"] = fmt.Sprintf("%s %s", versionInfo, now)

	configureOSOptions(conf)
}
