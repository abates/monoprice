package monoprice

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
)

var (
	ErrTooLong     = errors.New("String is too long")
	ErrReadTimeout = errors.New("Read Timeout")
	ErrInvalidZone = errors.New("Invalid Zone ID")
	ErrCommand     = errors.New("Invalid Command")

	Timeout = time.Second
)

type cmdResp struct {
	zone  int
	cmd   Command
	value string
	err   error
}

func (cr *cmdResp) Unmarshal(line string) (err error) {
	cr.zone, err = strconv.Atoi(line[0:2])
	if err != nil {
		return err
	}

	line = line[2:]
	cr.cmd = Command(line[0:2])
	cr.value = line[2:]
	if _, found := commands[cr.cmd]; !found {
		cr.cmd = ST
		cr.value = line
	}
	return nil
}

type subscription struct {
	unsubscribe bool
	zone        int
	cmd         Command
	ch          chan<- *cmdResp
}

type Amplifier struct {
	writer      io.Writer
	reader      *bufio.Reader
	writeCh     chan<- string
	subscribeCh chan<- *subscription
	zones       []Zone
}

func New(port io.ReadWriter) *Amplifier {
	writeCh := make(chan string)
	subscribeCh := make(chan *subscription)
	amp := &Amplifier{
		writer:      port,
		reader:      bufio.NewReader(port),
		writeCh:     writeCh,
		subscribeCh: subscribeCh,
	}

	cmdRespCh := make(chan *cmdResp)
	go amp.readLoop(cmdRespCh)
	go amp.writeLoop(cmdRespCh, subscribeCh, writeCh)

	return amp
}

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
	resp, err := amp.sendQuery(zone, ST)
	if err == nil {
		state.Unmarshal(resp)
	} else if err == ErrReadTimeout {
		err = ErrInvalidZone
	}
	return
}

func (z *zone) State() (state State, err error) {
	return z.amp.State(z.id)
}

func (z *zone) SendCommand(cmd Command, arg interface{}) error {
	_, err := z.amp.sendCmd(z.id, cmd, cmd.format(arg))
	return err
}

type Command string

func (c Command) format(v interface{}) string {
	return fmt.Sprintf(commands[c], v)
}

var (
	// Inquiries
	PA              Command = "PA"
	SetPower        Command = "PR"
	SetMute         Command = "MU"
	SetDND          Command = "DT"
	SetVolume       Command = "VO"
	SetTreble       Command = "TR"
	SetBass         Command = "BS"
	SetBalance      Command = "BL"
	SetSource       Command = "CH"
	GetKeypadStatus Command = "LS"
	ST              Command = "ST" // State is a pseudo command used only for conveying complete state in this API

	commands = map[Command]string{
		PA:              "",
		SetPower:        "%s",
		SetMute:         "%s",
		SetDND:          "%02d",
		SetVolume:       "%02d",
		SetTreble:       "%02d",
		SetBass:         "%02d",
		SetBalance:      "%02d",
		SetSource:       "%02d",
		GetKeypadStatus: "",
		ST:              "",
	}
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

func (amp *Amplifier) readLine() (line string, err error) {
	data, err := amp.reader.ReadBytes('\n')
	if err == nil {
		line = strings.TrimSpace(string(data))
	}
	return
}

func (amp *Amplifier) readLoop(respCh chan<- *cmdResp) {
	defer func() {
		close(respCh)
	}()

	lastCmd := Command("")
	for {
		line, err := amp.readLine()
		if err != nil {
			log.Printf("Failed to read line: %v", err)
			return
		}

		line = strings.TrimPrefix(line, "#")
		if strings.HasPrefix(line, ">") {
			// query response
			resp := &cmdResp{}
			line = strings.TrimPrefix(line, ">")
			err := resp.Unmarshal(line)
			if err == nil {
				respCh <- resp
			} else {
				log.Printf("Failed to parse response %q: %v", line, err)
			}
		} else if strings.HasPrefix(line, "?") || strings.HasPrefix(line, "!") {
			// query echo or command echo
			lastCmd = Command(line[1:3])
		} else if line == "Command Error." {
			respCh <- &cmdResp{cmd: lastCmd, err: ErrCommand}
		} else if len(line) > 0 {
			log.Printf("Unknown line format for %q", line)
		}
	}
}

func (amp *Amplifier) writeLoop(respCh <-chan *cmdResp, subscribeCh <-chan *subscription, writeCh <-chan string) {
	listeners := make(map[int]map[Command]map[chan<- *cmdResp]interface{})
	for {
		select {
		case req := <-subscribeCh:
			zone, found := listeners[req.zone]
			if !found {
				zone = make(map[Command]map[chan<- *cmdResp]interface{})
				listeners[req.zone] = zone
			}
			zl, found := zone[req.cmd]
			if !found {
				zl = make(map[chan<- *cmdResp]interface{})
				zone[req.cmd] = zl
			}
			if req.unsubscribe {
				close(req.ch)
				delete(zl, req.ch)
			} else {
				zl[req.ch] = nil
			}
		case resp := <-respCh:
			if zone, found := listeners[resp.zone]; found {
				if zl, found := zone[resp.cmd]; found {
					for ch := range zl {
						select {
						case ch <- resp:
						default:
							close(ch)
							delete(zl, ch)
						}
					}
				}
			}
		case cmd := <-writeCh:
			amp.writer.Write([]byte(cmd))
			amp.writer.Write([]byte("\r"))
		}
	}
}

func (amp *Amplifier) sendQuery(zone int, cmd Command) (string, error) {
	return amp.write(zone, "?", cmd, "")
}

func (amp *Amplifier) sendCmd(zone int, cmd Command, arg string) (string, error) {
	return amp.write(zone, "!", cmd, arg)
}

func (amp *Amplifier) write(zone int, typ string, cmd Command, arg string) (str string, err error) {
	ch := make(chan *cmdResp, 1)
	req := &subscription{
		zone: zone,
		cmd:  cmd,
		ch:   ch,
	}
	amp.subscribeCh <- req

	if cmd == ST {
		cmd = ""
	}

	if len(arg) > 0 {
		amp.writeCh <- fmt.Sprintf("%s%d%s", typ, zone, arg)
	} else {
		amp.writeCh <- fmt.Sprintf("%s%d%s%s", typ, zone, cmd, arg)
	}

	select {
	case <-time.After(Timeout):
		err = ErrReadTimeout
	case resp := <-ch:
		str = resp.value
		err = resp.err
	}

	req.unsubscribe = true
	amp.subscribeCh <- req
	return
}
