package monoprice

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
)

var (
	ErrInvalidZone     = errors.New("invalid Zone ID")
	ErrUnknownState    = errors.New("failed to determine state")
	ErrCommand         = errors.New("invalid Command")
	ErrInvalidResponse = errors.New("invalid response")
	ErrRetryTimeout    = errors.New("retries exceeded")

	QueryRetryLimit = 3
)

type Amplifier struct {
	writer     io.Writer
	reader     *bufio.Reader
	writeCh    chan<- cmdReq
	zones      []Zone
	verboseLog bool
}

type Option func(*Amplifier)

func VerboseOption() Option {
	return func(amp *Amplifier) {
		amp.verboseLog = true
	}
}

func New(port io.ReadWriter, options ...Option) (*Amplifier, error) {
	writeCh := make(chan cmdReq)
	amp := &Amplifier{
		writer:  port,
		reader:  bufio.NewReader(port),
		writeCh: writeCh,
	}

	for _, option := range options {
		option(amp)
	}

	go amp.writeLoop(writeCh)

	err := amp.initZones()
	return amp, err
}

func (amp *Amplifier) State(zone ZoneID) (state State, err error) {
	resp, err := amp.sendQuery(zone, ST)
	if err == nil {
		err = state.Unmarshal(resp)
	} else if err == io.EOF {
		err = ErrInvalidZone
	}
	return
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
			_, err := amp.State(id)
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
	return nil
}

func (amp *Amplifier) readLine() (line string, err error) {
	data, err := amp.reader.ReadBytes('\n')
	if err == nil {
		if amp.verboseLog {
			log.Printf("RX %s", string(data))
		}
		line = strings.TrimPrefix(strings.TrimSpace(string(data)), "#")
	} else if amp.verboseLog {
		log.Printf("RX Error: %v", err)
	}
	return line, err
}

func (amp *Amplifier) writeCommand(req *cmdReq) {
	maxTries := 3
	resp := cmdResp{}
	line := ""
	for tries := 0; tries < maxTries; tries++ {
		amp.writer.Write([]byte(req.cmd))
		amp.writer.Write([]byte("\r"))
		if amp.verboseLog {
			log.Printf("TX %s", req.cmd)
		}

		// wait for command to be echoed back
		line, resp.err = amp.readLine()
		if resp.err != nil {
			break
		}
		if line != req.cmd {
			resp.err = ErrInvalidResponse
			break
		}

		line, resp.err = amp.readLine()
		if resp.err != nil {
			break
		}

		if line == "" {
			if amp.verboseLog {
				log.Printf("Empty response received, re-sending command")
			}
			continue
		}

		if req.cmd[0:1] == "<" && resp.value == req.cmd {
			resp.err = resp.Unmarshal(line[1:])
			break
		}

		if req.cmd[0:1] == "?" && line[0:1] == ">" {
			resp.err = resp.Unmarshal(line[1:])
			break
		}

		resp.err = ErrRetryTimeout
	}

	req.resp <- resp
}

func (amp *Amplifier) writeLoop(writeCh <-chan cmdReq) {
	for req := range writeCh {
		amp.writeCommand(&req)
	}
}

func (amp *Amplifier) sendQuery(zone ZoneID, cmd Command) (string, error) {
	return amp.write(zone, "?", cmd, "")
}

func (amp *Amplifier) sendCmd(zone ZoneID, cmd Command, arg string) (string, error) {
	return amp.write(zone, "<", cmd, arg)
}

func (amp *Amplifier) write(zone ZoneID, typ string, cmd Command, arg string) (string, error) {
	ch := make(chan cmdResp, 1)
	if cmd == ST {
		cmd = ""
	}

	cmdStr := ""
	if len(arg) == 0 {
		cmdStr = fmt.Sprintf("%s%d%s", typ, zone, cmd)
	} else {
		cmdStr = fmt.Sprintf("%s%d%s%s", typ, zone, cmd, arg)
	}

	amp.writeCh <- cmdReq{cmd: cmdStr, resp: ch}
	resp := <-ch
	return resp.value, resp.err
}
