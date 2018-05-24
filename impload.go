package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	device = flag.String("d", "", "Serial Device")
	baud   = flag.Int("b", 115200, "Baud rate")
)


func do_test() {
	devname := check_device()
	MSPInit(devname, *baud)
}

func do_convert(inf string, outf string) {
	m, err := Read_Mission_File(inf)
	if m != nil && err == nil {
		m.Dump(outf)
	} else {
		log.Fatal("Invalid input file\n")
	}
}

func do_upload(inf string, eeprom bool) {
	devname := check_device()
	s := MSPInit(devname, *baud)
	m, err := Read_Mission_File(inf)
	if m != nil && err == nil {
		s.upload(m, eeprom)
	} else {
		log.Fatal("Invalid input file\n")
	}
}

func do_download(outf string, eeprom bool) {
	devname := check_device()
	s := MSPInit(devname, *baud)
	m := s.download(eeprom)
	m.Dump(outf)
}

func verify_in_out_files(files []string) (string, string) {
	var inf, outf string
	if len(files) == 0 {
		inf = "-"
		outf = "-"
	} else if len(files) == 1 {
		inf = files[0]
		outf = "-"
	} else {
		inf = files[0]
		outf = files[1]
	}
	return inf, outf
}

func check_device() string {
	devname := *device
	if devname == "" {
		for _, v := range []string{"/dev/ttyACM0", "/dev/ttyUSB0"} {
			if _, err := os.Stat(v); err == nil {
				devname = v
				break
			}
		}
	}
	if devname == "" {
		log.Fatalln("No device given\n")
	} else {
		log.Printf("Using device %s %d\n", devname, *baud)
	}
	return devname
}

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s [options] command [files ...]\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "  command\n\tAction required (upload|download|store|restore|convert)\n")
	}

	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		log.Fatal("No command given")
	}

	inf, outf := verify_in_out_files(files[1:])

	switch files[0] {
	case "test":
		do_test()
	case "upload", "up":
		do_upload(inf, false)
	case "download", "down":
		do_download(inf, false)
	case "convert", "conv":
		do_convert(inf, outf)
	case "store", "sto":
		do_upload(inf, true)
	case "restore", "rest":
		do_download(inf, true)
	}
}
