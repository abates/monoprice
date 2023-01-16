package monoprice

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
)

var (
	ErrInvalidZone     = errors.New("invalid Zone ID")
	ErrUnknownState    = errors.New("failed to determine state")
	ErrCommand         = errors.New("invalid Command")
	ErrInvalidResponse = errors.New("invalid response")
	ErrRetryTimeout    = errors.New("retries exceeded")
	ErrReadTimeout     = errors.New("read timeout")

	QueryRetryLimit = 3
)

type Amplifier struct {
	writer     io.Writer
	reader     *bufio.Reader
	zones      []Zone
	verboseLog bool
	mutex      sync.Mutex
	ignoreEOF  bool
}

type Option func(*Amplifier)

func VerboseOption() Option {
	return func(amp *Amplifier) {
		amp.verboseLog = true
	}
}

func New(port io.ReadWriter, options ...Option) (*Amplifier, error) {
	amp := &Amplifier{
		writer:    port,
		reader:    bufio.NewReader(port),
		ignoreEOF: false,
	}

	for _, option := range options {
		option(amp)
	}

	err := amp.initZones()
	return amp, err
}

func (amp *Amplifier) Zones() (zones []Zone) {
	return amp.zones
}

func (amp *Amplifier) initZones() error {
	log.Printf("Initializing amplifier zones")
	amp.zones = []Zone{}
	for i := 1; i < 4; i++ {
		for j := 1; j < 7; j++ {
			id := ZoneID(10*i + j)
			_, err := amp.QueryState(id)
			if err == nil {
				amp.zones = append(amp.zones, newZone(id, amp))
				log.Printf("Found Zone %d", id)
			} else if err == ErrInvalidZone {
				log.Printf("Zone %d is not attached", id)
			} else {
				return err
			}
		}
	}
	amp.ignoreEOF = true
	return nil
}

func (amp *Amplifier) readResponse() (string, error) {
	str, err := amp.reader.ReadString('#')
	if amp.verboseLog {
		log.Printf("RX %q (err: %v)", str, err)
	}
	return str, err
}

func (amp *Amplifier) write(cmdStr string, resp Response) error {
	amp.mutex.Lock()
	defer amp.mutex.Unlock()
	cmdStr = cmdStr + "\r\n"
	if amp.verboseLog {
		log.Printf("TX %q", cmdStr)
	}
	_, err := amp.writer.Write([]byte(cmdStr))
	if err == nil {
		err = resp.Read(amp)
	}
	if err == nil && resp.EchoString() != cmdStr {
		err = fmt.Errorf("%w wrong echo string, wanted %q got %q", ErrInvalidResponse, cmdStr, resp.EchoString())
	}
	return err
}

func (amp *Amplifier) QueryState(zone ZoneID) (State, error) {
	resp := &QueryResponse{}
	cmdStr := fmt.Sprintf("?%d", zone)
	err := amp.write(cmdStr, resp)
	return resp.State, err
}

func (amp *Amplifier) SendCommand(zone ZoneID, cmd Command, arg interface{}) error {
	argStr := cmd.format(arg)
	cmdStr := fmt.Sprintf("<%d%s%s", zone, cmd, argStr)
	resp := &EchoResponse{}
	return amp.write(cmdStr, resp)
}
