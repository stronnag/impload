package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"strconv"
)


const (
	DevClass_NONE = iota
	DevClass_SERIAL
	DevClass_TCP
	DevClass_UDP
	DevClass_BT
)

type DevDescription struct {
	klass  int
	name   string
	param  int
	name1  string
	param1 int
}

var (
	defalt     = flag.Int("a", 20, "Default altitude (m)")
	baud       = flag.Int("b", 115200, "Baud rate")
	device     = flag.String("d", "", "Serial Device")
	defspeed   = flag.Float64("s", 0, "Default speed (m/s)")
	force_rtl  = flag.Bool("force-rth", false, "Adds RTH for 'external' formats")
	force_land = flag.Bool("force-land", false, "Adds RTH / Land for 'external' formats")
	show_vers  = flag.Bool("v", false, "Shows version")
)


var GitCommit = "local"
var GitTag = "0.0.0"

func GetVersion() string {
	return fmt.Sprintf("impload %s, commit: %s", GitTag, GitCommit)
}

func do_test() {
	devdesc := check_device()
	MSPInit(devdesc)
}

func do_convert(inf string, outf string) {
	mtype, m, err := Read_Mission_File(inf)
	if m != nil && err == nil {
		sanitise_mission(m, mtype)
		m.Dump(outf, inf, mtype)
	} else {
		log.Fatal("Invalid input file\n")
	}
}

func sanitise_mission(m *Mission, mtype string) {
	for j, mi := range m.MissionItems {
		if mi.Action == "WAYPOINT" {
			if *defspeed != 0.0 && mi.P1 == 0 {
				m.MissionItems[j].P1 = int16(*defspeed * 100)
			}
			if mi.Alt == 0 {
				m.MissionItems[j].Alt = int32(*defalt)
			}
		}
	}
	if (mtype == "gpx" || mtype == "kml") && (*force_rtl || *force_land) {
		m.Add_rtl(*force_land)
	}
	if mlen := len(m.MissionItems); mlen > 60 {
		log.Fatal(fmt.Sprintf("Mission has too many (%d) waypoints\n", mlen))
	}
}

func do_upload(inf string, eeprom bool) {
	devdesc := check_device()
	s := MSPInit(devdesc)
	mtype, m, err := Read_Mission_File(inf)
	if m != nil && err == nil {
		sanitise_mission(m, mtype)
		s.upload(m, eeprom)
	} else {
		log.Fatal("Invalid input file\n")
	}
}

func do_download(outf string, eeprom bool) {
	devdesc := check_device()
	s := MSPInit(devdesc)
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

func check_device() DevDescription {
	devdesc := parse_device()
	if devdesc.name == "" {
		for _, v := range []string{"/dev/ttyACM0", "/dev/ttyUSB0"} {
			if _, err := os.Stat(v); err == nil {
				devdesc.klass = DevClass_SERIAL
				devdesc.name = v
				devdesc.param = *baud
				break
			}
		}
	}

	if devdesc.name == "" {
		log.Fatalln("No device available")
	} else {
		log.Printf("Using device [%v]\n", devdesc.name)
	}
	return devdesc
}

func parse_device() DevDescription {
	dd := DevDescription{name: "", klass: DevClass_NONE}
	r := regexp.MustCompile(`^(tcp|udp)://([\[\]:A-Za-z\-\.0-9]*):(\d+)/{0,1}([A-Za-z\-\.0-9]*):{0,1}(\d*)`)
	m := r.FindAllStringSubmatch(*device, -1)
	if len(m) > 0 {
		if m[0][1] == "tcp" {
			dd.klass = DevClass_TCP
		} else {
			dd.klass = DevClass_UDP
		}
		dd.name = m[0][2]
		dd.param, _ = strconv.Atoi(m[0][3])
		// These are only used for ESP8266 UDP
		dd.name1 = m[0][4]
		dd.param1, _ = strconv.Atoi(m[0][5])
	} else if len(*device) == 17 && (*device)[2] == ':' && (*device)[8] == ':' && (*device)[14] == ':' {
		dd.name = *device
		dd.klass = DevClass_BT
	} else {
		ss := strings.Split(*device, "@")
		dd.klass = DevClass_SERIAL
		dd.name = ss[0]
		if len(ss) > 1 {
			dd.param, _ = strconv.Atoi(ss[1])
		} else {
			dd.param = *baud
		}
	}
	return dd
}

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of impload [options] command [files ...]\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "  command\n\tAction required (upload|download|store|restore|convert|test)\n\n")
		fmt.Fprintln(os.Stderr, GetVersion())
	}

	flag.Parse()

	if *show_vers {
		fmt.Fprintf(os.Stderr, "%s\n", GitTag)
		os.Exit(0)
	}

	files := flag.Args()
	if len(files) == 0 {
		log.Fatal("No command given")
	}

	inf, outf := verify_in_out_files(files[1:])

	switch files[0] {
	case "help":
		flag.Usage()
		os.Exit(0)
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
	case "version":
		fmt.Fprintln(os.Stderr, GetVersion())
	default:
		fmt.Fprintf(os.Stderr, "impload: unrecognised command \"%s\"\n", files[0])
	}
}
