package driver_test

import (
	"testing"

	"ariga.io/atlas/cmd/atlas/cmd/internal/driver"
	"github.com/stretchr/testify/require"
)

func Test_ProviderNotSupported(t *testing.T) {
	_, err := driver.NewAtlas("fake://open")
	require.Error(t, err)
}

func Test_RegisterProvider(t *testing.T) {
	p := func(s string) (*driver.Atlas, error) { return nil, nil }
	require.NotPanics(t, func() { driver.Register("key", p) })
}

func Test_RegisterTwiceSameKeyFails(t *testing.T) {
	p := func(s string) (*driver.Atlas, error) { return nil, nil }
	require.NotPanics(t, func() { driver.Register("key", p) })
	require.Panics(t, func() { driver.Register("key", p) })
}
