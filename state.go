package monoprice

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	ErrTooLong = errors.New("String is too long")
)

type State struct {
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

type unmarshaler func(string) error

func intUnmarshaler(receiver *int) unmarshaler {
	return func(str string) (err error) {
		*receiver, err = strconv.Atoi(str)
		return err
	}
}

func boolUnmarshaler(receiver *bool) unmarshaler {
	return func(str string) (err error) {
		if str[0:1] != "0" {
			return strconv.ErrSyntax
		}
		*receiver, err = strconv.ParseBool(str[1:])
		return err
	}
}

func (state *State) Unmarshal(str string) (err error) {
	unmarshalers := []unmarshaler{
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
			err = ErrTooLong
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

type marshaler func() string

func intMarshaler(value int) marshaler {
	return func() string {
		return fmt.Sprintf("%02d", value)
	}
}

func boolMarshaler(value bool) marshaler {
	return func() string {
		if value {
			return fmt.Sprintf("01")
		}
		return fmt.Sprintf("00")
	}
}

func (state *State) Marshal() (string, error) {
	marshalers := []marshaler{
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
