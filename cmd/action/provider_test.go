package action

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ReDockerConfig(t *testing.T) {
	for _, tt := range []struct{ input, img, version, args string }{
		{"mysql", "mysql", "", ""},
		{"mysql:8", "mysql", "8", ""},
		{"mysql:8.0", "mysql", "8.0", ""},
		{"mysql:8.0-debian", "mysql", "8.0-debian", ""},
		{"mysql:8.0-debian?arg1=one&arg2=two", "mysql", "8.0-debian", "?arg1=one&arg2=two"},
		{"mariadb:10.8.2-rc-focal?port=3000&db=database", "mariadb", "10.8.2-rc-focal", "?port=3000&db=database"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			m := reDockerConfig.FindStringSubmatch(tt.input)
			require.Len(t, m, 4)
			require.Equal(t, []string{tt.img, tt.version, tt.args}, m[1:])
		})
	}
}
