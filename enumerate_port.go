package main

import (
	"errors"
	"go.bug.st/serial/enumerator"
	"os"
	"runtime"
)

func enumerate_ports() (string, error) {
	switch runtime.GOOS {
	case "linux", "windows":
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
		}
		return "", err
	case "freebsd":
		if _, err := os.Stat("/dev/cuaU0"); err == nil {
			return "/dev/cuaU0", nil
		}
	default:
		break
	}
	return "", errors.New("no port available")
}
