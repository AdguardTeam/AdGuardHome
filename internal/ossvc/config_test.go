package ossvc_test

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/ossvc"
	"github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureServiceOptions(t *testing.T) {
	t.Parallel()

	const (
		vInfo       = "v1.0.0"
		serviceName = "test-service"
		svcInfoKey  = "SvcInfo"
	)

	t.Run("nil_option", func(t *testing.T) {
		conf := &service.Config{
			Name: serviceName,
		}

		ossvc.ConfigureServiceOptions(conf, vInfo)
		require.NotNil(t, conf.Option)

		svcInfo, ok := conf.Option[svcInfoKey]
		require.True(t, ok)

		assert.Contains(t, svcInfo, vInfo)
	})

	t.Run("existing_option", func(t *testing.T) {
		t.Parallel()

		const (
			key = "ExistingKey"
			val = "ExistingValue"
		)

		conf := &service.Config{
			Name: serviceName,
			Option: map[string]any{
				key: val,
			},
		}

		ossvc.ConfigureServiceOptions(conf, vInfo)
		require.NotNil(t, conf.Option)

		assert.Equal(t, val, conf.Option[key])

		svcInfo, ok := conf.Option[svcInfoKey]
		require.True(t, ok)

		assert.Contains(t, svcInfo, vInfo)
	})
}
