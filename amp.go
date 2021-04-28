package monoprice

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

var (
	ErrReadTimeout = errors.New("Read Timeout")
	ErrInvalidZone = errors.New("Invalid Zone ID")
	ErrCommand     = errors.New("Invalid Command")

	Timeout = time.Second
)

type Amplifier struct {
	writer  io.Writer
	reader  *bufio.Reader
	writeCh chan<- cmdReq
	zones   map[ZoneID]*zone
}

func New(port io.ReadWriter) (*Amplifier, error) {
	writeCh := make(chan cmdReq)
	amp := &Amplifier{
		writer:  port,
		reader:  bufio.NewReader(port),
		writeCh: writeCh,
	}

	//go amp.readLoop()
	go amp.writeLoop(writeCh)

	err := amp.initZones()
	return amp, err
}

func (amp *Amplifier) State(zone ZoneID) (state State, err error) {
	resp, err := amp.sendQuery(zone, ST)
	if err == nil {
		err = state.Unmarshal(resp)
	} else if err == ErrReadTimeout {
		err = ErrInvalidZone
	}
	return
}

func (amp *Amplifier) Zones() (zones []Zone) {
	for _, zone := range amp.zones {
		zones = append(zones, zone)
	}
	return zones
}

func (amp *Amplifier) initZones() error {
	amp.zones = make(map[ZoneID]*zone)
	for i := 1; i < 4; i++ {
		for j := 1; j < 7; j++ {
			id := ZoneID(10*i + j)
			_, err := amp.State(id)
			if err == nil {
				amp.zones[id] = newZone(id, amp)
			} else if err != ErrInvalidZone {
				return err
			}
		}
	}
	return nil
}

func (amp *Amplifier) readLine() (line string, err error) {
	data, err := amp.reader.ReadBytes('\n')
	if err == nil {
		line = strings.TrimSpace(string(data))
	}
	return
}

func (amp *Amplifier) writeLoop(writeCh <-chan cmdReq) {
	for req := range writeCh {
		amp.writer.Write([]byte(req.cmd))
		amp.writer.Write([]byte("\r"))
		// wait for command to be echoed back
		for {
			line, err := amp.readLine()
			if err != nil {
				log.Fatalf("Failed to read: %v", err)
			}

			if req.cmd[0:1] == "<" && line == req.cmd {
				resp := cmdResp{}
				resp.err = resp.Unmarshal(line[1:])
				req.resp <- resp
				break
			} else if req.cmd[0:1] == "?" && line[1:2] == ">" {
				resp := cmdResp{}
				resp.err = resp.Unmarshal(line[1:])
				req.resp <- resp
				break
			}
		}
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
