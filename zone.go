package monoprice

import "errors"

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
	for i := 0; i < QueryRetryLimit; i++ {
		state, err = z.amp.QueryState(z.id)
		if err == nil || !errors.Is(ErrInvalidZone, err) {
			break
		}
	}
	if errors.Is(ErrInvalidZone, err) {
		err = ErrUnknownState
	}
	return
}

func (z *zone) SendCommand(cmd Command, arg interface{}) error {
	return z.amp.SendCommand(z.id, cmd, arg)
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
