package driver_test

import (
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/driver"
	"github.com/stretchr/testify/require"
)

func Test_ProviderNotSupported(t *testing.T) {
	_, err := driver.NewAtlas("fake://open")
	require.Error(t, err)
}

func Test_RegisterProvider(t *testing.T) {
	u := new(driver.URLMux)
	p := func(s string) (*driver.Atlas, error) { return nil, nil }
	require.NotPanics(t, func() { u.Providers.Register("key", p) })
}

func Test_RegisterTwiceSameKeyFails(t *testing.T) {
	u := new(driver.URLMux)
	p := func(s string) (*driver.Atlas, error) { return nil, nil }
	require.NotPanics(t, func() { u.Providers.Register("key", p) })
	require.Panics(t, func() { u.Providers.Register("key", p) })
}

func Test_GetDriverFails(t *testing.T) {
	_, err := driver.NewAtlas("key://open")
	require.Error(t, err)
}

func Test_GetDriverSuccess(t *testing.T) {
	u := new(driver.URLMux)
	p := func(s string) (*driver.Atlas, error) { return nil, nil }
	u.Providers.Register("key", p)
	_, err := driver.NewAtlas("key://open")
	require.NoError(t, err)
}
