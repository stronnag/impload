package main

import (
	"encoding/binary"
	"fmt"
	"go.bug.st/serial"
	"log"
	"net"
	"os"
	"time"
)

const (
	msp_API_VERSION = 1
	msp_FC_VARIANT  = 2
	msp_FC_VERSION  = 3
	msp_BOARD_INFO  = 4
	msp_BUILD_INFO  = 5

	msp_NAME            = 10
	msp_WP_MISSION_LOAD = 18
	msp_WP_MISSION_SAVE = 19
	msp_WP_GETINFO      = 20

	msp_WP     = 118
	msp_SET_WP = 209

	wp_BAD       = 182
	msp_DEBUGMSG = 253

	msp_COMMON_SETTING     = 0x1003
	msp_COMMON_SET_SETTING = 0x1004
	msp_EEPROM_WRITE       = 250
)

const (
	state_INIT = iota
	state_M
	state_DIRN
	state_LEN
	state_CMD
	state_DATA
	state_CRC

	state_X_HEADER2
	state_X_FLAGS
	state_X_ID1
	state_X_ID2
	state_X_LEN1
	state_X_LEN2
	state_X_DATA
	state_X_CHECKSUM
)

const (
	wp_WAYPOINT = 1 + iota
	wp_POSHOLD_UNLIM
	wp_POSHOLD_TIME
	wp_RTH
	wp_SET_POI
	wp_JUMP
	wp_SET_HEAD
	wp_LAND
)

const SETTING_STR string = "nav_wp_multi_mission_index"

type MsgData struct {
	ok   bool
	cmd  uint16
	len  uint16
	data []byte
}

type SerDev interface {
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	Close() error
}

type MSPSerial struct {
	klass int
	sd    SerDev
	c0    chan MsgData
}

var (
	Wp_count byte
	use_v2   bool
	dumphex  bool
)

func crc8_dvb_s2(crc byte, a byte) byte {
	crc ^= a
	for i := 0; i < 8; i++ {
		if (crc & 0x80) != 0 {
			crc = (crc << 1) ^ 0xd5
		} else {
			crc = crc << 1
		}
	}
	return crc
}

func encode_msp2(cmd uint16, payload []byte) []byte {
	var paylen int16
	if len(payload) > 0 {
		paylen = int16(len(payload))
	}
	buf := make([]byte, 9+paylen)
	buf[0] = '$'
	buf[1] = 'X'
	buf[2] = '<'
	buf[3] = 0 // flags
	binary.LittleEndian.PutUint16(buf[4:6], cmd)
	binary.LittleEndian.PutUint16(buf[6:8], uint16(paylen))
	if paylen > 0 {
		copy(buf[8:], payload)
	}
	crc := byte(0)
	for _, b := range buf[3 : paylen+8] {
		crc = crc8_dvb_s2(crc, b)
	}
	buf[8+paylen] = crc
	if dumphex {
		fmt.Fprintf(os.Stderr, "MSPV2 %d\n", cmd)
		hexdump(buf)
	}
	return buf
}

func encode_msp(cmd uint16, payload []byte) []byte {
	var paylen byte
	if len(payload) > 0 {
		paylen = byte(len(payload))
	}
	buf := make([]byte, 6+paylen)
	buf[0] = '$'
	buf[1] = 'M'
	buf[2] = '<'
	buf[3] = paylen
	buf[4] = byte(cmd)
	if paylen > 0 {
		copy(buf[5:], payload)
	}
	crc := byte(0)
	for _, b := range buf[3:] {
		crc ^= b
	}
	buf[5+paylen] = crc
	return buf
}

func (m *MSPSerial) Read_msp(c0 chan MsgData) {
	inp := make([]byte, 128)
	var sc MsgData
	var count = uint16(0)
	var crc = byte(0)

	n := state_INIT

	for {
		nb, err := m.sd.Read(inp)
		if err == nil && nb > 0 {
			for i := 0; i < nb; i++ {
				switch n {
				case state_INIT:
					if inp[i] == '$' {
						n = state_M
						sc.ok = false
						sc.len = 0
						sc.cmd = 0
					}
				case state_M:
					if inp[i] == 'M' {
						n = state_DIRN
					} else if inp[i] == 'X' {
						n = state_X_HEADER2
					} else {
						n = state_INIT
					}
				case state_DIRN:
					if inp[i] == '!' {
						n = state_LEN
					} else if inp[i] == '>' {
						n = state_LEN
						sc.ok = true
					} else {
						n = state_INIT
					}

				case state_X_HEADER2:
					if inp[i] == '!' {
						n = state_X_FLAGS
					} else if inp[i] == '>' {
						n = state_X_FLAGS
						sc.ok = true
					} else {
						n = state_INIT
					}

				case state_X_FLAGS:
					crc = crc8_dvb_s2(0, inp[i])
					n = state_X_ID1

				case state_X_ID1:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.cmd = uint16(inp[i])
					n = state_X_ID2

				case state_X_ID2:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.cmd |= (uint16(inp[i]) << 8)
					n = state_X_LEN1

				case state_X_LEN1:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.len = uint16(inp[i])
					n = state_X_LEN2

				case state_X_LEN2:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.len |= (uint16(inp[i]) << 8)
					if sc.len > 0 {
						n = state_X_DATA
						count = 0
						sc.data = make([]byte, sc.len)
					} else {
						n = state_X_CHECKSUM
					}
				case state_X_DATA:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.data[count] = inp[i]
					count++
					if count == sc.len {
						n = state_X_CHECKSUM
					}

				case state_X_CHECKSUM:
					ccrc := inp[i]
					if crc != ccrc {
						fmt.Fprintf(os.Stderr, "CRC error on %d\n", sc.cmd)
					} else {
						c0 <- sc
					}
					n = state_INIT

				case state_LEN:
					sc.len = uint16(inp[i])
					crc = inp[i]
					n = state_CMD
				case state_CMD:
					sc.cmd = uint16(inp[i])
					crc ^= inp[i]
					if sc.len == 0 {
						n = state_CRC
					} else {
						sc.data = make([]byte, sc.len)
						n = state_DATA
						count = 0
					}
				case state_DATA:
					sc.data[count] = inp[i]
					crc ^= inp[i]
					count++
					if count == sc.len {
						n = state_CRC
					}
				case state_CRC:
					ccrc := inp[i]
					if crc != ccrc {
						fmt.Fprintf(os.Stderr, "CRC error on %d\n", sc.cmd)
					} else {
						//						fmt.Fprintf(os.Stderr, "Cmd %v Len %v\n", sc.cmd, sc.len)
						c0 <- sc
					}
					n = state_INIT
				}
			}
		} else {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Read %v\n", err)
			} else {
				fmt.Fprintln(os.Stderr, "serial EOF")
			}
			m.sd.Close()
			os.Exit(2)
		}
	}
}

func NewMSPSerial(dd DevDescription) *MSPSerial {
	switch dd.klass {
	case DevClass_SERIAL:
		p, err := serial.Open(dd.name, &serial.Mode{BaudRate: dd.param})
		if err != nil {
			log.Fatal(err)
		}
		return &MSPSerial{klass: dd.klass, sd: p}
	case DevClass_BT:
		bt := NewBT(dd.name)
		return &MSPSerial{klass: dd.klass, sd: bt}
	case DevClass_TCP:
		var conn net.Conn
		remote := fmt.Sprintf("%s:%d", dd.name, dd.param)
		addr, err := net.ResolveTCPAddr("tcp", remote)
		if err == nil {
			conn, err = net.DialTCP("tcp", nil, addr)
		}
		if err != nil {
			log.Fatal(err)
		}
		return &MSPSerial{klass: dd.klass, sd: conn}
	case DevClass_UDP:
		var laddr, raddr *net.UDPAddr
		var conn net.Conn
		var err error
		if dd.param1 != 0 {
			raddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name1, dd.param1))
			laddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name, dd.param))
		} else {
			if dd.name == "" {
				laddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name, dd.param))
			} else {
				raddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name, dd.param))
			}
		}
		if err == nil {
			conn, err = net.DialUDP("udp", laddr, raddr)
		}
		if err != nil {
			log.Fatal(err)
		}
		return &MSPSerial{klass: dd.klass, sd: conn}
	default:
		fmt.Fprintln(os.Stderr, "Unsupported device")
		os.Exit(1)
	}
	return nil
}

func (m *MSPSerial) Send_msp(cmd uint16, payload []byte) {
	buf := encode_msp(cmd, payload)
	m.sd.Write(buf)
}

func (m *MSPSerial) Wait_msp(cmd uint16, payload []byte) MsgData {
	var buf []byte
	if use_v2 || cmd > 255 {
		buf = encode_msp2(cmd, payload)
	} else {
		buf = encode_msp(cmd, payload)
	}
	m.sd.Write(buf)

	var v MsgData
	for done := false; !done; {
		select {
		case v = <-m.c0:
			if v.cmd == cmd {
				done = true
			}
		case <-time.After(time.Second * 5):
			log.Fatalln("MSP timeout")
		}
	}
	return v
}

func MSPInit(dd DevDescription) *MSPSerial {
	var fw, api, vers, board, gitrev string

	dumphex = os.Getenv("IMPLOAD_DUMPHEX") != ""

	m := NewMSPSerial(dd)

	m.c0 = make(chan MsgData)
	go m.Read_msp(m.c0)

	m.Send_msp(msp_API_VERSION, nil)

	for done := false; !done; {
		select {
		case v := <-m.c0:
			switch v.cmd {
			case msp_API_VERSION:
				if v.len > 2 {
					api = fmt.Sprintf("%d.%d", v.data[1], v.data[2])
					use_v2 = (v.data[1] == 2)
					m.Send_msp(msp_FC_VARIANT, nil)
				}
			case msp_FC_VARIANT:
				fw = string(v.data[0:4])
				m.Send_msp(msp_FC_VERSION, nil)
			case msp_FC_VERSION:
				vers = fmt.Sprintf("%d.%d.%d", v.data[0], v.data[1], v.data[2])
				m.Send_msp(msp_BUILD_INFO, nil)
			case msp_BUILD_INFO:
				gitrev = string(v.data[19:])
				m.Send_msp(msp_BOARD_INFO, nil)
			case msp_BOARD_INFO:
				if v.len > 8 {
					board = string(v.data[9:])
				} else {
					board = string(v.data[0:4])
				}
				fmt.Printf("%s v%s %s (%s) API %s", fw, vers, board, gitrev, api)
				m.Send_msp(msp_NAME, nil)
			case msp_NAME:
				if v.len > 0 {
					fmt.Printf(" \"%s\"\n", v.data)
				} else {
					fmt.Println()
				}
				if Wp_count == 0 {
					z := make([]byte, 1)
					z[0] = 1
					m.Send_msp(msp_WP_MISSION_LOAD, z)
				} else {
					m.Send_msp(msp_WP_GETINFO, nil)
				}
			case msp_WP_MISSION_LOAD:
				m.Send_msp(msp_WP_GETINFO, nil)
			case msp_WP_GETINFO:
				wp_max := v.data[1]
				MaxWP = int(wp_max)
				wp_valid := v.data[2]
				Wp_count = v.data[3]
				fmt.Printf("Extant waypoints in FC: %d of %d, valid %d\n", Wp_count, wp_max, wp_valid)
				done = true
			default:
				fmt.Fprintf(os.Stderr, "Unsolicited %d, length %d\n", v.cmd, v.len)
			}
		}
	}
	return m
}

func Decode_action(b byte) string {
	var a string
	switch b {
	case wp_WAYPOINT:
		a = "WAYPOINT"
	case wp_POSHOLD_UNLIM:
		a = "POSHOLD_UNLIM"
	case wp_POSHOLD_TIME:
		a = "POSHOLD_TIME"
	case wp_RTH:
		a = "RTH"
	case wp_SET_POI:
		a = "SET_POI"
	case wp_JUMP:
		a = "JUMP"
	case wp_SET_HEAD:
		a = "SET_HEAD"
	case wp_LAND:
		a = "LAND"
	default:
		a = "UNKNOWN"
	}
	return a
}

func Encode_action(a string) byte {
	var b byte
	switch a {
	case "WAYPOINT":
		b = wp_WAYPOINT
	case "POSHOLD_UNLIM":
		b = wp_POSHOLD_UNLIM
	case "POSHOLD_TIME":
		b = wp_POSHOLD_TIME
	case "RTH":
		b = wp_RTH
	case "SET_POI":
		b = wp_SET_POI
	case "JUMP":
		b = wp_JUMP
	case "SET_HEAD":
		b = wp_SET_HEAD
	case "LAND":
		b = wp_LAND
	default:
		b = wp_WAYPOINT
	}
	return b
}

func serialise_wp(mi MissionItem, last bool) (int, []byte) {
	buf := make([]byte, 21)
	buf[0] = byte(mi.No)
	buf[1] = Encode_action(mi.Action)
	v := int32(mi.Lat * 1e7)
	binary.LittleEndian.PutUint32(buf[2:6], uint32(v))
	v = int32(mi.Lon * 1e7)
	binary.LittleEndian.PutUint32(buf[6:10], uint32(v))
	binary.LittleEndian.PutUint32(buf[10:14], uint32(100*mi.Alt))
	binary.LittleEndian.PutUint16(buf[14:16], uint16(mi.P1))
	binary.LittleEndian.PutUint16(buf[16:18], uint16(mi.P2))
	binary.LittleEndian.PutUint16(buf[18:20], uint16(mi.P3))
	buf[20] = mi.Flag
	if dumphex {
		fmt.Fprintf(os.Stderr, "WP %d\n", mi.No)
		hexdump(buf)
	}
	return len(buf), buf
}

func hexdump(buf []byte) {
	for _, b := range buf {
		fmt.Fprintf(os.Stderr, "%02x ", b)
	}
	fmt.Fprintln(os.Stderr)
}

func (m *MSPSerial) download(eeprom bool) *MultiMission {
	if eeprom {
		z := make([]byte, 1)
		z[0] = 1
		m.Wait_msp(msp_WP_MISSION_LOAD, z)
		fmt.Printf("Restored mission\n")
	}

	v := m.Wait_msp(msp_WP_GETINFO, nil)
	wp_count := v.data[3]
	var mis = []MissionItem{}
	if wp_count > 0 {
		z := make([]byte, 1)
		for z[0] = 1; ; z[0]++ {
			v := m.Wait_msp(msp_WP, z)
			if v.len > 0 {
				_, mi := deserialise_wp(v.data)
				mis = append(mis, mi)
				if z[0] == wp_count {
					break
				}
			}
		}
	}
	return NewMultiMission(mis)
}

func deserialise_wp(b []byte) (bool, MissionItem) {
	var lat, lon float64
	var action string
	var p1, p2, p3 int16
	var v, alt int32

	action = Decode_action(b[1])
	v = int32(binary.LittleEndian.Uint32(b[2:6]))
	lat = float64(v) / 1e7
	v = int32(binary.LittleEndian.Uint32(b[6:10]))
	lon = float64(v) / 1e7
	alt = int32(binary.LittleEndian.Uint32(b[10:14])) / 100
	p1 = int16(binary.LittleEndian.Uint16(b[14:16]))
	p2 = int16(binary.LittleEndian.Uint16(b[16:18]))
	p3 = int16(binary.LittleEndian.Uint16(b[18:20]))
	last := (b[20] == 0xa5)
	item := MissionItem{No: int(b[0]), Lat: lat, Lon: lon, Alt: alt, Action: action, P1: p1, P2: p2, P3: p3, Flag: b[20]}
	if *verbose {
		fmt.Printf("D: %d %d\n", b[0], b[20])
	}
	return last, item
}

func (s *MSPSerial) upload(mm *MultiMission, eeprom bool) {
	if mm.is_valid() {

		i := 0
		for _, ms := range mm.Segment {
			mlen := len(ms.MissionItems)
			for _, v := range ms.MissionItems {
				i++
				v.No = i
				if *verbose == false {
					fmt.Printf("Upload %d\r", i)
				}
				_, b := serialise_wp(v, (i == mlen))
				if *verbose {
					fmt.Fprintf(os.Stderr, "Buf %d %d %d --- ", b[0], b[20], v.Flag)
				}
				s.Wait_msp(msp_SET_WP, b)
				if *verbose {
					fmt.Fprintf(os.Stderr, "Buf %d %d\n", b[0], b[20])
				}
			}
		}
		fmt.Printf("upload %d, save %v\n", i, eeprom)

		if eeprom {
			z := make([]byte, 1)
			z[0] = 1
			t := time.Now()
			s.Wait_msp(msp_WP_MISSION_SAVE, z)
			et := time.Since(t)
			fmt.Printf("Saved mission (%s)\n", et)
		}
		v := s.Wait_msp(msp_WP_GETINFO, nil)
		wp_max := v.data[1]
		wp_valid := v.data[2]
		wp_count := v.data[3]
		fmt.Printf("Waypoints: %d of %d, valid %d\n", wp_count, wp_max, wp_valid)
	} else {
		fmt.Printf("Mission fails verification, upload cancelled\n")
	}
}

func (m *MSPSerial) get_multi_index() {
	lstr := len(SETTING_STR)
	buf := make([]byte, lstr+1)
	copy(buf, SETTING_STR)
	buf[lstr] = 0
	v := m.Wait_msp(msp_COMMON_SETTING, buf)
	if v.len > 0 {
		fmt.Printf("Multi index %d\n", v.data[0])
	}
}

func (m *MSPSerial) set_multi_index(idx uint8) {
	lstr := len(SETTING_STR)
	buf := make([]byte, lstr+2)
	copy(buf, SETTING_STR)
	buf[lstr] = 0
	buf[lstr+1] = idx
	m.Wait_msp(msp_COMMON_SET_SETTING, buf)
	m.Wait_msp(msp_EEPROM_WRITE, nil)
}
