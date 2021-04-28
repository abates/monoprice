package monoprice

import (
	"fmt"
	"strconv"
)

type cmdReq struct {
	cmd  string
	resp chan<- cmdResp
}

type cmdResp struct {
	zone  ZoneID
	cmd   Command
	value string
	err   error
}

func (cr *cmdResp) Unmarshal(line string) (err error) {
	zone, err := strconv.Atoi(line[0:2])
	if err != nil {
		return err
	}
	cr.zone = ZoneID(zone)
	line = line[2:]
	cr.cmd = Command(line[0:2])
	if _, found := commands[cr.cmd]; !found {
		cr.cmd = ST
		cr.value = line
	} else {
		cr.value = line[2:]
	}
	return nil
}

type Command string

func (c Command) format(v interface{}) string {
	return fmt.Sprintf(commands[c], v)
}

type commandSequence struct {
	zoneID    ZoneID
	direction string
	cmd       Command
	args      string
}

func (cs *commandSequence) Marshal() string {
	if len(cs.args) == 0 {
		return fmt.Sprintf("%s%d%s", cs.direction, cs.zoneID, cs.cmd)
	}
	return fmt.Sprintf("%s%d%s%s", cs.direction, cs.zoneID, cs.cmd, cs.args)
}

var (
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
