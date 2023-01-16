package monoprice

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type State struct {
	Zone         int  `json:"zone"`
	PA           bool `json:"pa"`
	Power        bool `json:"power"`
	Mute         bool `json:"mute"`
	DoNotDisturb bool `json:"do_not_disturb"`
	Volume       int  `json:"volume"`
	Treble       int  `json:"treble"`
	Bass         int  `json:"bass"`
	Balance      int  `json:"balance"`
	Source       int  `json:"source"`
	KeyPad       bool `json:"keypad"`
}

func (state *State) Unmarshal(str string) (err error) {
	unmarshalers := []unmarshaler{
		intUnmarshaler(&state.Zone),
		boolUnmarshaler(&state.PA),
		boolUnmarshaler(&state.Power),
		boolUnmarshaler(&state.Mute),
		boolUnmarshaler(&state.DoNotDisturb),
		intUnmarshaler(&state.Volume),
		intUnmarshaler(&state.Treble),
		intUnmarshaler(&state.Bass),
		intUnmarshaler(&state.Balance),
		intUnmarshaler(&state.Source),
		boolUnmarshaler(&state.KeyPad),
	}

	for err == nil {
		if len(str) < 2 {
			err = io.ErrUnexpectedEOF
		} else if len(unmarshalers) == 0 {
			err = fmt.Errorf("%w trailing %q", ErrTooLong, str)
		} else {
			err = unmarshalers[0](str[0:2])
			if err == nil {
				str = str[2:]
				unmarshalers = unmarshalers[1:]
				if len(unmarshalers) == 0 && len(str) == 0 {
					break
				}
			}
		}
	}
	return err
}

func (state *State) Marshal() (string, error) {
	marshalers := []marshaler{
		intMarshaler(state.Zone),
		boolMarshaler(state.PA),
		boolMarshaler(state.Power),
		boolMarshaler(state.Mute),
		boolMarshaler(state.DoNotDisturb),
		intMarshaler(state.Volume),
		intMarshaler(state.Treble),
		intMarshaler(state.Bass),
		intMarshaler(state.Balance),
		intMarshaler(state.Source),
		boolMarshaler(state.KeyPad),
	}

	builder := &strings.Builder{}
	for _, marshaler := range marshalers {
		builder.WriteString(marshaler())
	}
	return builder.String(), nil
}

type ampReader interface {
	readResponse() (string, error)
}

type Response interface {
	Read(ampReader) error
	EchoString() string
}

type EchoResponse struct {
	Echo string
}

func (er *EchoResponse) Read(reader ampReader) (err error) {
	er.Echo, err = reader.readResponse()
	if err == nil {
		er.Echo = strings.TrimRight(er.Echo, "#")
	}
	return err
}

func (er *EchoResponse) EchoString() string {
	return er.Echo
}

type QueryResponse struct {
	EchoResponse
	State State
}

func (qr *QueryResponse) Read(reader ampReader) error {
	err := qr.EchoResponse.Read(reader)
	stateString := ""
	if err == nil {
		stateString, err = reader.readResponse()
		if err == nil {
			if len(stateString) > 0 && stateString[0] == '>' {
				stateString = strings.TrimSpace(strings.TrimRight(stateString, "#"))
				err = qr.State.Unmarshal(stateString[1:])
			} else {
				err = fmt.Errorf("%w received %q", ErrInvalidResponse, stateString)
			}
		} else if errors.Is(err, io.EOF) {
			err = ErrInvalidZone
		}
	}
	return err
}
