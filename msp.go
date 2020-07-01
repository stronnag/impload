package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"go.bug.st/serial"
	"log"
	"net"
	"os"
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
)
const (
	state_INIT = iota
	state_M
	state_DIRN
	state_LEN
	state_CMD
	state_DATA
	state_CRC
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

type MsgData struct {
	cmd  byte
	len  int
	data []byte
}

type MSPSerial struct {
	klass  int
	p      serial.Port
	conn   net.Conn
	reader *bufio.Reader
	bt     *BTConn
	c0     chan MsgData
}

func encode_msp(cmd byte, payload []byte) []byte {
	var paylen byte
	if len(payload) > 0 {
		paylen = byte(len(payload))
	}
	buf := make([]byte, 6+paylen)
	buf[0] = '$'
	buf[1] = 'M'
	buf[2] = '<'
	buf[3] = paylen
	buf[4] = cmd
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

func (m *MSPSerial) read(inp []byte) (int, error) {
	switch m.klass {
	case DevClass_SERIAL:
		return m.p.Read(inp)
	case DevClass_TCP:
		return m.conn.Read(inp)
	case DevClass_UDP:
		return m.reader.Read(inp)
	case DevClass_BT:
		return m.bt.Read(inp)
	}
	return -1, nil
}

func (m *MSPSerial) write(payload []byte) (int, error) {
	switch m.klass {
	case DevClass_SERIAL:
		return m.p.Write(payload)
	case DevClass_BT:
		return m.bt.Write(payload)
	default:
		return m.conn.Write(payload)
	}
}

func (m *MSPSerial) Read_msp(c0 chan MsgData) {
	inp := make([]byte, 128)
	var count = byte(0)
	var len = byte(0)
	var crc = byte(0)
	var cmd = byte(0)
	var sc MsgData

	n := state_INIT

	for {
		nb, err := m.read(inp)
		if err == nil {
			for i := 0; i < nb; i++ {
				switch n {
				case state_INIT:
					if inp[i] == '$' {
						n = state_M
						count = 0
						len = 0
						crc = 0
						cmd = 0
					}
				case state_M:
					if inp[i] == 'M' {
						n = state_DIRN
					} else {
						n = state_INIT
					}
				case state_DIRN:
					if inp[i] == '!' {
						n = state_LEN
					} else if inp[i] == '>' {
						n = state_LEN
					} else {
						n = state_INIT
					}
				case state_LEN:
					len = inp[i]
					crc = len
					count = 0
					n = state_CMD
				case state_CMD:
					cmd = inp[i]
					crc ^= cmd
					if len == 0 {
						n = state_CRC
					} else {
						n = state_DATA
					}
					sc.len = int(len)
					sc.data = make([]byte, sc.len)
					sc.cmd = cmd
				case state_DATA:
					sc.data[count] = inp[i]
					crc ^= inp[i]
					count++
					if count == len {
						n = state_CRC
					}
				case state_CRC:
					ccrc := inp[i]
					if crc != ccrc {
						fmt.Fprintf(os.Stderr, "CRC error on %d\n", cmd)
					} else {
						c0 <- sc
					}
					n = state_INIT
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "Read err\n")
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
		return &MSPSerial{klass: dd.klass, p: p}
	case DevClass_BT:
		bt := NewBT(dd.name)
		return &MSPSerial{klass: dd.klass, bt: bt}
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
		return &MSPSerial{klass: dd.klass, conn: conn}
	case DevClass_UDP:
		var laddr, raddr *net.UDPAddr
		var reader *bufio.Reader
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
			if err == nil {
				reader = bufio.NewReader(conn)
			}
		}
		if err != nil {
			log.Fatal(err)
		}
		return &MSPSerial{klass: dd.klass, conn: conn, reader: reader}
	default:
		fmt.Fprintln(os.Stderr, "Unsupported device")
		os.Exit(1)
	}
	return nil
}

func (m *MSPSerial) Send_msp(cmd byte, payload []byte) {
	buf := encode_msp(cmd, payload)
	m.write(buf)
}

func (m *MSPSerial) Wait_msp(cmd byte, payload []byte) MsgData {
	buf := encode_msp(cmd, payload)
	m.write(buf)

	var v MsgData
	for done := false; !done; {
		select {
		case v = <-m.c0:
			if v.cmd == cmd {
				done = true
			}
		}
	}
	return v
}

func MSPInit(dd DevDescription) *MSPSerial {
	var fw, api, vers, board, gitrev string
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
				fmt.Fprintf(os.Stderr, "%s v%s %s (%s) API %s", fw, vers, board, gitrev, api)
				m.Send_msp(msp_NAME, nil)
			case msp_NAME:
				if v.len > 0 {
					fmt.Fprintf(os.Stderr, " \"%s\"\n", v.data)
				} else {
					fmt.Fprintln(os.Stderr, "")
				}
				m.Send_msp(msp_WP_GETINFO, nil)
			case msp_WP_GETINFO:
				wp_max := v.data[1]
				wp_valid := v.data[2]
				wp_count := v.data[3]
				fmt.Fprintf(os.Stderr, "Extant waypoints in FC: %d of %d, valid %d\n", wp_count, wp_max, wp_valid)
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

func encode_action(a string) byte {
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
	buf := make([]byte, 32)
	buf[0] = byte(mi.No)
	buf[1] = encode_action(mi.Action)
	v := int32(mi.Lat * 1e7)
	binary.LittleEndian.PutUint32(buf[2:6], uint32(v))
	v = int32(mi.Lon * 1e7)
	binary.LittleEndian.PutUint32(buf[6:10], uint32(v))
	binary.LittleEndian.PutUint32(buf[10:14], uint32(100*mi.Alt))
	binary.LittleEndian.PutUint16(buf[14:16], uint16(mi.P1))
	binary.LittleEndian.PutUint16(buf[16:18], uint16(mi.P2))
	binary.LittleEndian.PutUint16(buf[18:20], uint16(mi.P3))
	if last {
		buf[20] = 0xa5
	} else {
		buf[20] = 0
	}
	return len(buf), buf
}

func (m *MSPSerial) download(eeprom bool) (ms *Mission) {
	if eeprom {
		z := make([]byte, 1)
		z[0] = 1
		m.Wait_msp(msp_WP_MISSION_LOAD, z)
		fmt.Fprintf(os.Stderr, "Restored mission\n")
	}

	var last bool
	z := make([]byte, 1)
	s := GetVersion()
	items := []MissionItem{}
	mission := &Mission{s, items}
	for z[0] = 1; !last; z[0]++ {
		v := m.Wait_msp(msp_WP, z)
		if v.len > 0 {
			l, mi := deserialise_wp(v.data)
			last = l
			mission.MissionItems = append(mission.MissionItems, mi)
		}
	}
	return mission
}

func deserialise_wp(b []byte) (bool, MissionItem) {
	var lat, lon float64
	var action string
	var p1, p2 int16
	var p3 uint16
	var v, alt int32

	action = Decode_action(b[1])
	v = int32(binary.LittleEndian.Uint32(b[2:6]))
	lat = float64(v) / 1e7
	v = int32(binary.LittleEndian.Uint32(b[6:10]))
	lon = float64(v) / 1e7
	alt = int32(binary.LittleEndian.Uint32(b[10:14])) / 100
	p1 = int16(binary.LittleEndian.Uint16(b[14:16]))
	p2 = int16(binary.LittleEndian.Uint16(b[16:18]))
	p3 = binary.LittleEndian.Uint16(b[18:20])
	last := (b[20] == 0xa5)
	item := MissionItem{No: int(b[0]), Lat: lat, Lon: lon, Alt: alt, Action: action, P1: p1, P2: p2, P3: p3}
	return last, item
}

func (m *MSPSerial) upload(ms *Mission, eeprom bool) {

	if ms.is_valid() {
		mlen := len(ms.MissionItems)
		fmt.Fprintf(os.Stderr, "upload %d, save %v\n", mlen, eeprom)
		for i, v := range ms.MissionItems {
			fmt.Fprintf(os.Stderr, "Upload %d\r", i)
			_, b := serialise_wp(v, (i == mlen-1))
			m.Wait_msp(msp_SET_WP, b)
		}

		if eeprom {
			z := make([]byte, 1)
			z[0] = 1
			m.Wait_msp(msp_WP_MISSION_SAVE, z)
			fmt.Fprintf(os.Stderr, "Saved mission\n")
		}
		v := m.Wait_msp(msp_WP_GETINFO, nil)
		wp_max := v.data[1]
		wp_valid := v.data[2]
		wp_count := v.data[3]
		fmt.Fprintf(os.Stderr, "Waypoints: %d of %d, valid %d\n", wp_count, wp_max, wp_valid)
	} else {
		fmt.Fprintf(os.Stderr, "Mission fails verification, upload cancelled\n")
	}
}
