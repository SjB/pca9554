// +build ignore
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/SjB/pca9554"
	"github.com/kidoman/embd"

	_ "github.com/kidoman/embd/host/all"
)

// INT connected to GPIO_17 of RPI
// PCA9554A_GPIO_0 = button
// PCA9554A_GPIO_1 = button
// PCA9554A_GPIO_2 .. PCA9554A_GPIO_7 connected to LED

func main() {
	flag.Parse()

	embd.SetHost(embd.HostRPi, 2)

	if err := embd.InitI2C(); err != nil {
		panic(err)
	}
	defer embd.CloseI2C()

	if err := embd.InitGPIO(); err != nil {
		panic(err)
	}
	defer embd.CloseGPIO()

	bus := embd.NewI2CBus(1)
	fmt.Println("connected to bus")
	pca9554a := pca9554.New(bus, 0x38)

	if err := pca9554a.WriteConfiguration(0x00); err != nil {
		panic(err)
	}

	irqPin, err := embd.NewDigitalPin("GPIO_17")
	if err != nil {
		panic(err)
	}

	irq := make(chan byte)
	if err := pca9554a.SetInteruptPin(irqPin, func(b byte) { irq <- b }); err != nil {
		panic(err)
	}

	led0, err := pca9554a.DigitalPin("GPIO_6")
	if err != nil {
		panic(err)
	}
	led0.SetDirection(embd.Out)

	led1, err := pca9554a.DigitalPin("GPIO_7")
	if err != nil {
		panic(err)
	}
	led1.SetDirection(embd.Out)

	button0, err := pca9554a.DigitalPin("GPIO_0")
	if err != nil {
		panic(err)
	}
	button0.ActiveLow(true)

	button0.Watch(embd.EdgeNone, func(p embd.DigitalPin) {
		b, err := p.Read()
		if err != nil {
			panic(err)
		}
		v := toggle(led0)
		fmt.Printf("Button %v: [%#02x] Led0 %d\n", p, b, v)
	})

	button1, err := pca9554a.DigitalPin("GPIO_1")
	if err != nil {
		panic(err)
	}
	button1.ActiveLow(true)

	button1.Watch(embd.EdgeNone, func(p embd.DigitalPin) {
		b, err := p.Read()
		if err != nil {
			panic(err)
		}
		v := toggle(led1)
		fmt.Printf("Button %v: [%#02x] Led1 %d\n", p, b, v)
	})

	conf, err := pca9554a.ReadConfiguration()
	if err != nil {
		panic(err)
	}
	fmt.Printf("configuration %#02x\n", conf)

	pol, err := pca9554a.ReadPolarityInversionReg()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Polarity %#02x\n", pol)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	timer := time.Tick(2 * time.Second)

	i := 0
	var b byte
	for {
		i = (i + 1) % 16
		select {
		case b = <-irq:
			fmt.Printf("IRQ %#02x\n", b)
			continue
		case <-timer:
			outReg, err := pca9554a.ReadOutputReg()
			if err != nil {
				break
			}
			fmt.Printf("OutputReg: %#02X %#02X\n", outReg, outReg&0xC2)
			pca9554a.WriteOutputReg((byte(i) << 2) | (outReg & 0xC2))
		case <-c:
			pca9554a.Close()
			return
		}
	}
}

func toggle(p embd.DigitalPin) int {
	v, _ := p.Read()
	if v == embd.Low {
		v = embd.High
	} else {
		v = embd.Low
	}
	p.Write(v)
	return v
}
