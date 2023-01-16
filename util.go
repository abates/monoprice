package monoprice

import (
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrTooLong = errors.New("string is too long")
)

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

type marshaler func() string

func intMarshaler(value int) marshaler {
	return func() string {
		return fmt.Sprintf("%02d", value)
	}
}

func boolMarshaler(value bool) marshaler {
	return func() string {
		if value {
			return "01"
		}
		return "00"
	}
}
