package mux_test

import (
	"testing"

	"ariga.io/atlas/cmd/atlas/internal/mux"
	"github.com/stretchr/testify/require"
)

func Test_ProviderNotSupported(t *testing.T) {
	u := mux.NewMux()
	_, err := u.OpenAtlas("fake://open")
	require.Error(t, err)
}

func Test_RegisterProvider(t *testing.T) {
	u := mux.NewMux()
	p := func(s string) (*mux.Driver, error) { return nil, nil }
	require.NotPanics(t, func() { u.RegisterProvider("key", p) })
}

func Test_RegisterTwiceSameKeyFails(t *testing.T) {
	u := mux.NewMux()
	p := func(s string) (*mux.Driver, error) { return nil, nil }
	require.NotPanics(t, func() { u.RegisterProvider("key", p) })
	require.Panics(t, func() { u.RegisterProvider("key", p) })
}

func Test_GetDriverFails(t *testing.T) {
	u := mux.NewMux()
	_, err := u.OpenAtlas("key://open")
	require.Error(t, err)
}

func Test_GetDriverSuccess(t *testing.T) {
	u := mux.NewMux()
	p := func(s string) (*mux.Driver, error) { return nil, nil }
	u.RegisterProvider("key", p)
	_, err := u.OpenAtlas("key://open")
	require.NoError(t, err)
}
