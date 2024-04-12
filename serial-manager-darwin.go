//go:build darwin
// +build darwin

package main

import (
	"errors"
	"go.bug.st/serial"
	"log"
)

func enumerate_ports() (string, error) {
	return "", errors.New("Port name required on MacOS")
}

func open_serial_port(dd DevDescription) *MSPSerial {
	p, err := serial.Open(dd.name, &serial.Mode{BaudRate: dd.param})
	if err != nil {
		log.Fatal(err)
	}
	return &MSPSerial{packet: false, sd: p}

}
