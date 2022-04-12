package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	DevClass_NONE = iota
	DevClass_SERIAL
	DevClass_TCP
	DevClass_UDP
	DevClass_BT
)

const INAV_MAX_WP = 255

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
	outfmt     = flag.String("fmt", "xml", "Output format (xml, json, cli, xml-ugly)")
	MaxWP      = 60
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
		//		sanitise_mission(m, mtype)
		m.Dump(*outfmt, outf, inf, mtype)
	} else {
		log.Fatal("Invalid input file\n")
	}
}

func sanitise_mission(mm *MultiMission, mtype string) {
	for _, m := range mm.Segment {
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
	}
}

func do_clear(eeprom bool) {
	devdesc := check_device()
	s := MSPInit(devdesc)
	mis := []MissionItem{}
	item := MissionItem{No: 1, Lat: 0.0, Lon: 0.0, Alt: int32(25), Action: "RTH", Flag: 0xa5}
	mis = append(mis, item)
	mm := NewMultiMission(mis)
	s.upload(mm, eeprom)
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
	m.Dump(*outfmt, outf)
}

func do_get_multi_index() {
	devdesc := check_device()
	s := MSPInit(devdesc)
	s.get_multi_index()
}

func do_set_multi_index(mval int) {
	devdesc := check_device()
	s := MSPInit(devdesc)
	s.set_multi_index(uint8(mval))
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

	if devdesc.name == "" && devdesc.klass == DevClass_SERIAL {
		log.Fatalln("No device available")
	} else {
		log.Printf("Using device [%v]\n", devdesc.name)
	}
	return devdesc
}

func resolve_default_gw() string {
	cmds := []string{"ip route show 0.0.0.0/0 | cut -d ' ' -f3",
		"route -n | grep UG | awk '{print $2}'",
		"route -n show  0.0.0.0 | grep gateway | awk '{print $2}'"}

	ostr := os.Getenv("MWP_SERIAL_HOST")
	if ostr != "" {
		return ostr
	}
	for _, c := range cmds {
		out, err := exec.Command("sh", "-c", c).Output()
		ostr := strings.TrimSpace(string(out))
		if err != nil {
			log.Fatal(err)
		} else {
			if len(ostr) > 0 {
				return ostr
			}
		}
	}
	return "__MWP_SERIAL_HOST"
}

func parse_device() DevDescription {
	dd := DevDescription{name: "", klass: DevClass_NONE}
	r := regexp.MustCompile(`^(tcp|udp)://(__MWP_SERIAL_HOST|[\[\]:A-Za-z\-\.0-9]*):(\d+)/{0,1}([A-Za-z\-\.0-9]*):{0,1}(\d*)`)
	m := r.FindAllStringSubmatch(*device, -1)
	if len(m) > 0 {
		if m[0][1] == "tcp" {
			dd.klass = DevClass_TCP
		} else {
			dd.klass = DevClass_UDP
		}
		dd.name = m[0][2]
		if dd.name == "__MWP_SERIAL_HOST" {
			dd.name = resolve_default_gw()
		}
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
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "  command:\n\tAction required (upload|download|store|restore|convert|test|clear|erase|multi[=n])\n\n")
		fmt.Fprintln(os.Stderr, GetVersion())
	}

	flag.Parse()

	if *show_vers {
		fmt.Fprintf(os.Stderr, "%s\n", GitTag)
		os.Exit(0)
	}

	files := flag.Args()
	if len(files) == 0 {
		flag.Usage()
		os.Exit(-1)
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
	case "clear", "erase":
		do_clear(files[0] == "erase")
	case "multi":
		do_get_multi_index()
	case "version":
		fmt.Fprintln(os.Stderr, GetVersion())
	default:
		if strings.HasPrefix(files[0], "multi=") {
			mparts := strings.Split(files[0], "=")
			if len(mparts) == 2 {
				mval, err := strconv.Atoi(mparts[1])
				if err == nil && mval >= 0 && mval < 10 {
					do_set_multi_index(mval)
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "impload: unrecognised command \"%s\"\n", files[0])
		}
	}
}
