package main

import (
	"strconv"
	"strings"
	"syscall"
	"log"
	"golang.org/x/sys/unix"
)

func str2ba(addr string) [6]byte {
	a := strings.Split(addr, ":")
	var b [6]byte
	for i, tmp := range a {
		u, _ := strconv.ParseUint(tmp, 16, 8)
		b[len(b)-1-i] = byte(u)
	}
	return b
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Connect_bt(id string) int {
	mac := str2ba(id)
	fd, err := unix.Socket(syscall.AF_BLUETOOTH, syscall.SOCK_STREAM, unix.BTPROTO_RFCOMM)
	check(err)
	addr := &unix.SockaddrRFCOMM{Addr: mac, Channel: 1}
	err = unix.Connect(fd, addr)
	check(err)
	return fd
}

func Read_bt(fd int, buf []byte) (int, error) {
	n, err := unix.Read(fd, buf)
	return n, err
}

func Write_bt(fd int, buf []byte) (int, error) {
	n, err := unix.Write(fd, buf)
	return n, err
}

func Close_bt(fd int) {
	unix.Close(fd)
}

/**
func main() {
	var id string
	if len(os.Args) > 1 {
		id = os.Args[1]
	} else {
		log.Fatal("no device given")
	}
	fd := Connect_bt(id)
	defer Close_bt(fd)

	var buf = make([]byte, 128)
	for {
		n, err := Read_bt(fd, buf)
		check(err)
		if n > 0 {
			fmt.Printf("Received: %v\n", string(buf[0:n]))
		}
	}
}
**/
