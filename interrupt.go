// Package pca9554 interrupt subsystem
package pca9554

import "github.com/kidoman/embd"

type interrupt struct {
	pin     embd.DigitalPin
	handler func(embd.DigitalPin)
}

type interruptListener struct {
	interruptablePins map[embd.DigitalPin]func(embd.DigitalPin)
}

func defaultInterruptListener() *interruptListener {
	return &interruptListener{interruptablePins: make(map[embd.DigitalPin]func(embd.DigitalPin), 8)}
}

func (l *interruptListener) handle(b byte) {
	for p, h := range l.interruptablePins {
		if pin, ok := p.(*digitalPin); ok {
			if (pin.pin & b) != 0x0 {
				h(p)
			}
		}
	}
}

func (l *interruptListener) registerInterrupt(pin embd.DigitalPin, handler func(embd.DigitalPin)) {
	l.interruptablePins[pin] = handler
}

func (l *interruptListener) unregisterInterrupt(pin embd.DigitalPin) {
	delete(l.interruptablePins, pin)
}
