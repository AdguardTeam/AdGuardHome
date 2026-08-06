package ossvc

import (
	"fmt"
	"time"

	"github.com/kardianos/service"
)

// ConfigureServiceOptions defines additional settings of the service
// configuration.  conf must not be nil.
//
// TODO(e.burkov):  Use [timeutil.Clock].
func ConfigureServiceOptions(conf *service.Config, versionInfo string) {
	if conf.Option == nil {
		conf.Option = map[string]any{}
	}

	conf.Option["SvcInfo"] = fmt.Sprintf("%s %s", versionInfo, time.Now())

	configureOSOptions(conf)
}
