package pca9554

import (
	"errors"
	"fmt"
	"time"

	"github.com/kidoman/embd"
)

var pins = embd.PinMap{
	&embd.PinDesc{ID: "IO0", Aliases: []string{"0", "GPIO_0"}, Caps: embd.CapDigital, DigitalLogical: 0},
	&embd.PinDesc{ID: "IO1", Aliases: []string{"1", "GPIO_1"}, Caps: embd.CapDigital, DigitalLogical: 1},
	&embd.PinDesc{ID: "IO2", Aliases: []string{"2", "GPIO_2"}, Caps: embd.CapDigital, DigitalLogical: 2},
	&embd.PinDesc{ID: "IO3", Aliases: []string{"3", "GPIO_3"}, Caps: embd.CapDigital, DigitalLogical: 3},
	&embd.PinDesc{ID: "IO4", Aliases: []string{"4", "GPIO_4"}, Caps: embd.CapDigital, DigitalLogical: 4},
	&embd.PinDesc{ID: "IO5", Aliases: []string{"5", "GPIO_5"}, Caps: embd.CapDigital, DigitalLogical: 5},
	&embd.PinDesc{ID: "IO6", Aliases: []string{"6", "GPIO_6"}, Caps: embd.CapDigital, DigitalLogical: 6},
	&embd.PinDesc{ID: "IO7", Aliases: []string{"7", "GPIO_7"}, Caps: embd.CapDigital, DigitalLogical: 7},
}

type digitalPin struct {
	device *PCA9554
	id     string
	n      int
	mask   byte
	pin    byte
}

func (d *PCA9554) DigitalPin(key interface{}) (embd.DigitalPin, error) {
	pd, found := pins.Lookup(key, embd.CapDigital)
	if !found {
		return nil, fmt.Errorf("gpio: could not find pin matching %v", key)
	}

	pin := byte(0x1 << uint(pd.DigitalLogical))
	return &digitalPin{
		device: d,
		id:     pd.ID,
		n:      pd.DigitalLogical,
		pin:    pin,
		mask:   0xff ^ pin,
	}, nil
}

func (p *digitalPin) Watch(edge embd.Edge, handler func(embd.DigitalPin)) error {
	if err := p.SetDirection(embd.In); err != nil {
		return err
	}
	p.device.listener.registerInterrupt(p, handler)
	return nil
}

func (p *digitalPin) StopWatching() error {
	p.device.listener.unregisterInterrupt(p)
	return nil
}

func (p *digitalPin) N() int {
	return p.n
}

func (p *digitalPin) Write(val int) error {
	pin := byte(0)
	if val != embd.Low {
		pin = byte(p.pin)
	}

	reg := (p.device.outputPortReg & p.mask) | pin
	return p.device.WriteOutputReg(reg)
}

func (p *digitalPin) Read() (int, error) {
	reg, err := p.device.ReadInputReg()
	if err != nil {
		return 0, err
	}
	if 0 == (reg &^ p.mask) {
		return embd.Low, nil
	}
	return embd.High, nil
}

func (p *digitalPin) TimePulse(state int) (time.Duration, error) {
	aroundState := embd.Low
	if state == embd.Low {
		aroundState = embd.High
	}

	// Wait for any previous pulse to end
	for {
		v, err := p.Read()
		if err != nil {
			return 0, err
		}

		if v == aroundState {
			break
		}
	}

	// Wait until ECHO goes high
	for {
		v, err := p.Read()
		if err != nil {
			return 0, err
		}

		if v == state {
			break
		}
	}

	startTime := time.Now() // Record time when ECHO goes high

	// Wait until ECHO goes low
	for {
		v, err := p.Read()
		if err != nil {
			return 0, err
		}

		if v == aroundState {
			break
		}
	}

	return time.Since(startTime), nil // Calculate time lapsed for ECHO to transition from high to low
}

func (p *digitalPin) SetDirection(dir embd.Direction) error {
	reg := byte(0)
	if embd.In == dir {
		reg = p.pin
	}

	return p.device.WriteConfiguration(p.device.configurationReg&p.mask | reg)
}

func (p *digitalPin) ActiveLow(b bool) error {
	state := byte(0)
	if b {
		state = p.pin
	}
	return p.device.WritePolarityInversionReg(p.device.polarityInvReg&p.mask | state)
}

func (p *digitalPin) PullUp() error {
	return errors.New("gpio: not implemented")
}

func (p *digitalPin) PullDown() error {
	return errors.New("gpio: not implemented")
}

func (p *digitalPin) Close() error {
	return p.StopWatching()
}
