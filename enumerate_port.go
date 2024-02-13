package main

import (
	"fmt"
	"github.com/albenik/go-serial/enumerator"
	"os"
	"runtime"
)

func enumerate_ports() (string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err == nil {
		for _, port := range ports {
			if port.Name != "" {
				if port.IsUSB {
					if port.VID == "0483" && port.PID == "5740" ||
						port.VID == "0403" && port.PID == "6001" {
						return port.Name, nil
					}
				}
			}
		}
	} else {
		if runtime.GOOS == "freebsd" {
			for j := 0; j < 10; j++ {
				name := fmt.Sprintf("/dev/cuaU%d", j)
				if _, serr := os.Stat(name); serr == nil {
					return name, nil
				}
			}
		}
	}
	return "", err
}
