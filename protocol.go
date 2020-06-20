package monoprice

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

var (
	ErrTooLong      = errors.New("String is too long")
	ErrReadTimeout  = errors.New("Read Timeout")
	ErrWriteTimeout = errors.New("Write Timeout")
	ErrInvalidZone  = errors.New("Invalid Zone ID")
	ErrCommand      = errors.New("Invalid Command")
)

type writeResponse struct {
	line string
	err  error
}

type writeRequest struct {
	cmd  string
	resp chan *writeResponse
}

type Amplifier struct {
	writer io.Writer
	reader *bufio.Reader

	readCh  chan string
	writeCh chan *writeRequest

	zones []Zone
}

func New(port io.ReadWriter) *Amplifier {
	return &Amplifier{
		writer: port,
		reader: bufio.NewReader(port),
	}
}

type State struct {
	Zone         int  `json:"zone"`
	PA           bool `json:"pa"`
	Power        bool `json:"power"`
	Mute         bool `json:"mute"`
	DoNotDisturb bool `json:"do_not_disturb"`
	Volume       int  `json:"volume"`
	Treble       int  `json:"balance"`
	Bass         int  `json:"balance"`
	Balance      int  `json:"balance"`
	Source       int  `json:"source"`
	KeyPad       bool `json:"keypad"`
}

type unmarshaler func(string) error

func ParseBool(str string) (interface{}, error) {
	b := false
	err := boolUnmarshaler(&b)(str)
	return b, err
}

func ParseInt(str string) (interface{}, error) {
	i := 0
	err := intUnmarshaler(&i)(str)
	return i, err
}

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

type Zone interface {
	ID() int
	State() (State, error)
	SendCommand(cmd Command, arg interface{}) error
}

type zone struct {
	id  int
	amp *Amplifier
}

func (z *zone) ID() int {
	return z.id
}

func (amp *Amplifier) State(zone int) (state State, err error) {
	resp, err := amp.write(fmt.Sprintf("?%d", zone))
	if err == nil {
		if len(resp) == 0 {
			err = ErrInvalidZone
		} else {
			state.Unmarshal(resp)
		}
	}
	return
}

func (z *zone) State() (state State, err error) {
	return z.amp.State(z.id)
}

func (z *zone) SendCommand(cmd Command, arg interface{}) error {
	_, err := z.amp.write(fmt.Sprintf("<%d%s", z.id, cmd.format(arg)))
	return err
}

type Command string

func (c Command) format(v interface{}) string {
	return fmt.Sprintf(string(c), v)
}

var (
	SetPower   Command = "PR%s"
	SetMute    Command = "MU%s"
	SetVolume  Command = "VO%02d"
	SetTreble  Command = "TR%02d"
	SetBass    Command = "BS%02d"
	SetBalance Command = "BL%02d"
	SetSource  Command = "CH%02d"
)

func (z *zone) Restore(state State) (err error) {
	for _, cmd := range []struct {
		cmd Command
		arg interface{}
	}{
		{SetPower, boolMarshaler(state.Power)},
		{SetMute, boolMarshaler(state.Mute)},
		{SetVolume, state.Volume},
		{SetTreble, state.Treble},
		{SetBass, state.Bass},
		{SetBalance, state.Balance},
		{SetSource, state.Source},
	} {
		err = z.SendCommand(cmd.cmd, cmd.arg)
		if err != nil {
			break
		}
	}
	return err
}

func (amp *Amplifier) Zones() ([]Zone, error) {
	if amp.zones == nil {
		for i := 1; i < 4; i++ {
			for j := 1; j < 7; j++ {
				id := 10*i + j
				_, err := amp.State(id)
				if err == nil {
					amp.zones = append(amp.zones, &zone{amp: amp, id: id})
				} else if err != ErrInvalidZone {
					return nil, err
				}
			}
		}
	}
	return amp.zones, nil
}

func (amp *Amplifier) readLoop() {
	for {
		line, _, err := amp.reader.ReadLine()
		if err == nil {
			amp.readCh <- strings.TrimPrefix(string(line), "#")
		} else {
			break
		}
	}
	close(amp.readCh)
}

func (amp *Amplifier) writeLoop() {
	var req *writeRequest
	for {
		select {
		case line := <-amp.readCh:
			if req != nil {
				// ignore echos
				if line == req.cmd {
					continue
				} else if strings.HasPrefix(line, "Command Error") {
					resp := &writeResponse{err: ErrCommand}
					req.resp <- resp
				} else {
					resp := &writeResponse{line: line}
					req.resp <- resp
				}
				close(req.resp)
				req = nil
			}
		case req = <-amp.writeCh:
			amp.writer.Write([]byte(req.cmd))
			amp.writer.Write([]byte("\r"))
		}
	}
}

func (amp *Amplifier) write(cmd string) (str string, err error) {
	resp := make(chan *writeResponse, 1)
	timeout := 5 * time.Second
	select {
	case <-time.After(timeout):
		err = ErrWriteTimeout
	case amp.writeCh <- &writeRequest{cmd, resp}:
		select {
		case <-time.After(timeout):
			err = ErrReadTimeout
		case r := <-resp:
			str = r.line
			err = r.err
		}
	}
	return
}
