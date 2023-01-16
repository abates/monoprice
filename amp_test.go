package monoprice

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func TestAmpQueryState(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"Good", "?11\r\n#>1100000000130705100301\r\r\n#", nil},
	}

	// 2023/01/15 20:16:02 RX "?11\r\n#" (err: <nil>)
	// 2023/01/15 20:16:02 RX ">1100000000130705100301\r\r\n#" (err: <nil>)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			amp := Amplifier{
				reader: bufio.NewReader(strings.NewReader(test.input)),
				writer: io.Discard,
			}
			_, gotErr := amp.QueryState(11)
			if test.wantErr != gotErr {
				t.Errorf("Wanted error %v got %v", test.wantErr, gotErr)
			}
		})
	}
}
