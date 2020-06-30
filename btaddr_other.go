// +build !linux

package main

import (
	"log"
	"errors"
)

func Connect_bt(id string) int {
	log.Fatal("BT sockets are Linux only")
	return -1
}

func Read_bt(fd int, buf []byte) (int, error) {
	return -1, errors.New("Unsupported OS")
}

func Write_bt(fd int, buf []byte) (int, error) {
	return -1, errors.New("Unsupported OS")
}
func Close_bt(fd int) {
}
