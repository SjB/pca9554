// Package pca9554 allows interfacing with the pca9554 8-bit I2C I/O expansion chip.
package pca9554

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/kidoman/embd"
)

const (
	InputPortRegister = iota
	OutputPortRegister
	PolarityInvRegister
	ConfigurationRegister
)

// PCA9554(A) 8-bit I2C I/O expansion chip
type PCA9554 struct {
	Bus  embd.I2CBus
	Addr byte

	// Cached chip states
	configurationReg byte
	polarityInvReg   byte
	inputPortReg     byte
	outputPortReg    byte

	initialized bool

	// reference to the interrupt pin.
	interruptPin embd.DigitalPin

	listener *interruptListener
	mu       sync.RWMutex
}

// New creates a new PCA9554 interface
func New(bus embd.I2CBus, addr byte) *PCA9554 {
	return &PCA9554{
		Bus:              bus,
		Addr:             addr,
		configurationReg: 0xff, // Default Register 3 settings
		// regPolInv: Default Register 2 settings are 0x00

		listener: defaultInterruptListener(),
	}
}

// Set the GPIO Pin that the PCA9554 INT pin is connected to.
func (d *PCA9554) SetInteruptPin(pin embd.DigitalPin, handler func(byte)) error {
	if d.interruptPin != nil {
		// only one interrupt pin on this product
		return fmt.Errorf("pca9554: interrupt pin has already been set to %v", d.interruptPin.N())
	}

	if err := pin.SetDirection(embd.In); err != nil {
		return err
	}

	// only listen to Falling edge since the interrupt pin is active low
	err := pin.Watch(embd.EdgeFalling, func(p embd.DigitalPin) {
		i, err := d.ReadInputReg()
		if err != nil {
			glog.Fatal("pca9554: can't read input register")
		}
		b := (i & d.configurationReg)
		handler(b)
		d.listener.handle(b)
	})
	if err != nil {
		return err
	}

	d.interruptPin = pin
	return nil
}

// We need to disconnect from the Interrupt pin.
func (d *PCA9554) Close() error {
	if nil == d.interruptPin {
		return nil
	}

	if err := d.interruptPin.Close(); err != nil {
		return err
	}

	d.interruptPin = nil
	return nil
}

// Write pin direction configuration (Register 3)
// 1 for embd.Out
// 0 for embd.In
// each bit in the byte represent one port.
func (d *PCA9554) WriteConfiguration(conf byte) error {
	glog.V(1).Infof("pca9554: new GPIO configuration [%#02x]", conf)
	if err := d.Bus.WriteByteToReg(d.Addr, ConfigurationRegister, conf); err != nil {
		return err
	}
	d.configurationReg = conf
	return nil
}

// Read pin direction configuration (Register 3)
// each bit in the byte represent one port.
func (d *PCA9554) ReadConfiguration() (byte, error) {
	conf, err := d.Bus.ReadByteFromReg(d.Addr, ConfigurationRegister)
	if err != nil {
		return 0, err
	}
	glog.V(1).Infof("pca9554: current GPIO configuration [%#02x]", conf)
	d.configurationReg = conf
	return conf, nil
}

// Write the polarity inversion configuration into Register 2
// 0 = Input Port register data retained
// 1 = Input Port register data is inverted
// each bit in the byte represent one port.
func (d *PCA9554) WritePolarityInversionReg(reg byte) error {
	glog.V(1).Infof("pca9554: write polarity invsersion settings [%#02x]", reg)
	if err := d.Bus.WriteByteToReg(d.Addr, PolarityInvRegister, reg); err != nil {
		return err
	}
	d.polarityInvReg = reg
	return nil
}

// Read current polarity inversion register (Register 2) into a byte
// Each bit in the byte represent one port.
func (d *PCA9554) ReadPolarityInversionReg() (byte, error) {
	reg, err := d.Bus.ReadByteFromReg(d.Addr, PolarityInvRegister)
	if err != nil {
		return 0, err
	}
	glog.V(1).Infof("pca9554: current polarity inversion settings [%#02x]", reg)
	d.polarityInvReg = reg
	return reg, nil
}

// Read input register (Register 0) into a byte
// Each bit in the byte represent one port.
func (d *PCA9554) ReadInputReg() (byte, error) {
	b, err := d.Bus.ReadByteFromReg(d.Addr, InputPortRegister)
	if err != nil {
		return 0, err
	}
	glog.V(1).Infof("pca9554: reading [%#02x] from input", b)
	d.inputPortReg = b
	return b, nil
}

// Write byte to output register (Register 1)
// Each bit in the byte represent one port.
func (d *PCA9554) WriteOutputReg(b byte) error {
	glog.V(1).Infof("pca9554: writing [%#02x] to pins", b)
	if err := d.Bus.WriteByteToReg(d.Addr, OutputPortRegister, b); err != nil {
		return err
	}
	d.outputPortReg = b
	return nil
}

func (d *PCA9554) ReadOutputReg() (byte, error) {
	b, err := d.Bus.ReadByteFromReg(d.Addr, OutputPortRegister)
	if err != nil {
		return 0, err
	}
	glog.V(1).Infof("pca9554: reading [%#02x] from output", b)
	d.outputPortReg = b
	return b, nil
}
