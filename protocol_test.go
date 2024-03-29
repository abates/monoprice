package monoprice

import (
	"bufio"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func Test_intUnmarshaler(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr error
	}{
		{"test 1", "01", 1, nil},
		{"test 2", "02", 2, nil},
		{"test 3", "03", 3, nil},
		{"test 4", "04", 4, nil},
		{"test 5", "5", 5, nil},
		{"test 6", "foo", 0, strconv.ErrSyntax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := 0
			gotErr := intUnmarshaler(&got)(tt.input)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("Wanted error %v got %T", tt.wantErr, gotErr)
			} else if gotErr == nil {
				if got != tt.want {
					t.Errorf("Wanted %d got %d", tt.want, got)
				}
			}
		})
	}
}

func Test_boolUnmarshaler(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr error
	}{
		{"test 1", "01", true, nil},
		{"test 2", "00", false, nil},
		{"test 3", "10", false, strconv.ErrSyntax},
		{"test 4", "foo", false, strconv.ErrSyntax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := false
			gotErr := boolUnmarshaler(&got)(tt.input)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("Wanted error %v got %T", tt.wantErr, gotErr)
			} else if gotErr == nil {
				if got != tt.want {
					t.Errorf("Wanted %v got %v", tt.want, got)
				}
			}
		})
	}
}

func Test_Marshaler(t *testing.T) {
	tests := []struct {
		name  string
		input marshaler
		want  string
	}{
		{"test 1", intMarshaler(1), "01"},
		{"test 2", intMarshaler(2), "02"},
		{"test 3", intMarshaler(3), "03"},
		{"test 4", intMarshaler(11), "11"},
		{"test 5", boolMarshaler(true), "01"},
		{"test 6", boolMarshaler(false), "00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input()
			if tt.want != got {
				t.Errorf("Wanted %q got %q", tt.want, got)
			}
		})
	}
}

func TestState_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    State
		wantErr error
	}{
		{"test 1", "1100010000131112100401", State{11, false, true, false, false, 13, 11, 12, 10, 4, true}, nil},
		{"test 2", "110001000010111210040", State{}, io.ErrUnexpectedEOF},
		{"test 3", "1177010000101112100401", State{}, strconv.ErrSyntax},
		{"test 4", "11000100dfsf112100401", State{}, strconv.ErrSyntax},
		{"test 5", "", State{}, io.ErrUnexpectedEOF},
		{"test 6", "110001000013111210040110", State{}, ErrTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := State{}
			gotErr := got.Unmarshal(tt.input)
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("Wanted error %v got %v", tt.wantErr, gotErr)
			} else if gotErr == nil {
				if tt.want != got {
					t.Errorf("Wanted %+v got %+v", tt.want, got)
				}
			}
		})
	}
}

func Test_intMarshaler(t *testing.T) {
	tests := []struct {
		name  string
		input State
		want  string
	}{
		{"test 1", State{11, false, true, false, false, 13, 11, 12, 10, 4, true}, "1100010000131112100401"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.input.Marshal()
			if tt.want != got {
				t.Errorf("Wanted %q got %q", tt.want, got)
			}
		})
	}
}

func Test_boolMarshaler(t *testing.T) {
	type args struct {
		value bool
	}
	tests := []struct {
		name string
		args args
		want marshaler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := boolMarshaler(tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("boolMarshaler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_Marshal(t *testing.T) {
	type fields struct {
		Zone         int
		PA           bool
		Power        bool
		Mute         bool
		DoNotDisturb bool
		Volume       int
		Treble       int
		Bass         int
		Balance      int
		Source       int
		KeyPad       bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stat := &State{
				PA:           tt.fields.PA,
				Power:        tt.fields.Power,
				Mute:         tt.fields.Mute,
				DoNotDisturb: tt.fields.DoNotDisturb,
				Volume:       tt.fields.Volume,
				Treble:       tt.fields.Treble,
				Bass:         tt.fields.Bass,
				Balance:      tt.fields.Balance,
				Source:       tt.fields.Source,
				KeyPad:       tt.fields.KeyPad,
			}
			got, err := stat.Marshal()
			if (err != nil) != tt.wantErr {
				t.Errorf("State.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("State.Marshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testReader struct {
	*bufio.Reader
}

func (tr testReader) readResponse() (string, error) {
	return tr.ReadString('#')
}

func TestEchoResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    EchoResponse
		wantErr error
	}{
		{"Good echo", "SomeCmd\r#", EchoResponse{"SomeCmd\r"}, nil},
		{"Bad echo", "SomeCmd\r", EchoResponse{"SomeCmd\r"}, io.EOF},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := EchoResponse{}
			reader := testReader{bufio.NewReader(strings.NewReader(test.input))}
			gotErr := got.Read(reader)
			if test.want != got {
				t.Errorf("Wanted %+v but got %+v", test.want, got)
			}
			if test.wantErr != gotErr {
				t.Errorf("Wanted error %v but got %v", test.wantErr, gotErr)
			}
		})
	}
}

func TestQueryResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    QueryResponse
		wantErr error
	}{
		{
			name:  "Good echo",
			input: "?11\r#>1100000000130705100301\r#",
			want: QueryResponse{
				EchoResponse{"?11\r"},
				State{11, false, false, false, false, 13, 07, 05, 10, 03, true},
			},
			wantErr: nil,
		},
		{
			name:  "Bad echo",
			input: "?11\r##",
			want: QueryResponse{
				EchoResponse{"?11\r"},
				State{},
			},
			wantErr: ErrInvalidResponse,
		},
		{
			name:  "Invalid Zone",
			input: "?11\r#",
			want: QueryResponse{
				EchoResponse{"?11\r"},
				State{},
			},
			wantErr: ErrInvalidZone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := QueryResponse{}
			reader := testReader{bufio.NewReader(strings.NewReader(test.input))}
			gotErr := got.Read(reader)
			if test.want != got {
				t.Errorf("Wanted %+v but got %+v", test.want, got)
			}
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Wanted error %v but got %v", test.wantErr, gotErr)
			}
		})
	}
}
