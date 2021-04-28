package monoprice

type ZoneID int

type Zone interface {
	ID() ZoneID
	State() (State, error)
	SendCommand(cmd Command, arg interface{}) error
}

type zone struct {
	//*Events
	id  ZoneID
	amp *Amplifier
}

func newZone(id ZoneID, amp *Amplifier) *zone {
	return &zone{
		//Events: newEvents(),
		id:  id,
		amp: amp,
	}
}

func (z *zone) ID() ZoneID {
	return z.id
}

func (z *zone) State() (state State, err error) {
	return z.amp.State(z.id)
}

func (z *zone) SendCommand(cmd Command, arg interface{}) error {
	_, err := z.amp.sendCmd(z.id, cmd, cmd.format(arg))
	return err
}

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
