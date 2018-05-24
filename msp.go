package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/tarm/serial"
	"log"
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

	wp_WAYPOINT = 1
	wp_RTH      = 4

	state_INIT = iota
	state_M
	state_DIRN
	state_LEN
	state_CMD
	state_DATA
	state_CRC
)

type MSPSerial struct {
	p *serial.Port
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

func (m *MSPSerial) Read_msp() (byte, []byte, error) {
	inp := make([]byte, 1)
	var count = byte(0)
	var len = byte(0)
	var crc = byte(0)
	var cmd = byte(0)
	ok := true
	done := false
	var buf []byte

	n := state_INIT

	for !done {
		_, err := m.p.Read(inp)
		if err == nil {
			switch n {
			case state_INIT:
				if inp[0] == '$' {
					n = state_M
				}
			case state_M:
				if inp[0] == 'M' {
					n = state_DIRN
				} else {
					n = state_INIT
				}
			case state_DIRN:
				if inp[0] == '!' {
					n = state_LEN
					ok = false
				} else if inp[0] == '>' {
					n = state_LEN
				} else {
					n = state_INIT
				}
			case state_LEN:
				len = inp[0]
				buf = make([]byte, len)
				crc = len
				n = state_CMD
			case state_CMD:
				cmd = inp[0]
				crc ^= cmd
				if len == 0 {
					n = state_CRC
				} else {
					n = state_DATA
				}
			case state_DATA:
				buf[count] = inp[0]
				crc ^= inp[0]
				count++
				if count == len {
					n = state_CRC
				}
			case state_CRC:
				ccrc := inp[0]
				if crc != ccrc {
					ok = false
				}
				done = true
			}
		}
	}
	if !ok {
		return 0, nil, errors.New("MSP error")
	} else {
		return cmd, buf, nil
	}
}

func NewMSPSerial(name string, baud int) *MSPSerial {
	c := &serial.Config{Name: name, Baud: baud}
	p, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	return &MSPSerial{p}
}

func (m *MSPSerial) Send_msp(cmd byte, payload []byte) {
	buf := encode_msp(cmd, payload)
	m.p.Write(buf)
}

func MSPInit(devnam string, baud int) *MSPSerial {
	var fw, api, vers, board, gitrev string

	m := NewMSPSerial(devnam, baud)

	m.Send_msp(msp_API_VERSION, nil)
	_, payload, err := m.Read_msp()
	if err != nil {
		fmt.Println("read: ", err)
	} else {
		api = fmt.Sprintf("%d.%d", payload[1], payload[2])
	}

	m.Send_msp(msp_FC_VARIANT, nil)
	_, payload, err = m.Read_msp()
	if err != nil {
		fmt.Println("read: ", err)
	} else {
		fw = string(payload[0:4])
	}

	m.Send_msp(msp_FC_VERSION, nil)
	_, payload, err = m.Read_msp()
	if err != nil {
		fmt.Println("read: ", err)
	} else {
		vers = fmt.Sprintf("%d.%d.%d", payload[0], payload[1], payload[2])
	}

	m.Send_msp(msp_BUILD_INFO, nil)
	_, payload, err = m.Read_msp()
	if err != nil {
		fmt.Println("read: ", err)
	} else {
		gitrev = string(payload[19:])
	}

	m.Send_msp(msp_BOARD_INFO, nil)
	_, payload, err = m.Read_msp()
	if err != nil {
		fmt.Println("read: ", err)
	} else {
		board = string(payload[9:])
	}

	fmt.Printf("%s v%s %s (%s) API %s", fw, vers, board, gitrev, api)

	m.Send_msp(msp_NAME, nil)
	_, payload, err = m.Read_msp()

	if len(payload) > 0 {
		fmt.Printf(" \"%s\"\n", payload)
	} else {
		fmt.Println("")
	}

	m.Send_msp(msp_WP_GETINFO, nil)
	_, payload, err = m.Read_msp()
	if err != nil {
		fmt.Println("read: ", err)
	} else {
		wp_max := payload[1]
		wp_valid := payload[2]
		wp_count := payload[3]
		fmt.Printf("Waypoints: %d of %d, valid %d\n", wp_count, wp_max, wp_valid)
	}
	return m
}

func serialise_wp(mi MissionItem, last bool) (int, []byte) {
	buf := make([]byte, 32)
	buf[0] = byte(mi.No)
	if mi.Action == "RTH" {
		buf[1] = wp_RTH
	} else {
		buf[1] = wp_WAYPOINT
	}
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
		m.Send_msp(msp_WP_MISSION_LOAD, z)
		_, _, err := m.Read_msp()
		if err != nil {
			fmt.Printf("failed to restore mission: %s\n", err)
		} else {
			fmt.Printf("Restored mission\n")
		}
	}

	var last bool
	z := make([]byte, 1)
	version := Version{Value: "inept 0.0"}
	items := []MissionItem{}
	mission := &Mission{Version: version, MissionItems: items}
	for z[0] = 1; !last; z[0]++ {
		m.Send_msp(msp_WP, z)
		_, payload, err := m.Read_msp()
		if err == nil {
			l, mi := deserialise_wp(payload)
			last = l
			mission.MissionItems = append(mission.MissionItems, mi)
		}
	}
	return mission
}

func deserialise_wp(b []byte) (bool, MissionItem) {
	var lat, lon float64
	var action string
	var p1 int16
	var v, alt int32

	if b[1] == wp_RTH {
		action = "RTH"
	} else {
		action = "WAYPOINT"
	}
	v = int32(binary.LittleEndian.Uint32(b[2:6]))
	lat = float64(v) / 1e7
	v = int32(binary.LittleEndian.Uint32(b[6:10]))
	lon = float64(v) / 1e7
	alt = int32(binary.LittleEndian.Uint32(b[10:14]))/100
	p1 = int16(binary.LittleEndian.Uint16(b[14:16]))
	last := (b[20] == 0xa5)
	item := MissionItem{No: int(b[0]), Lat: lat, Lon: lon, Alt: alt, Action: action, P1: p1}
	return last, item
}

func (m *MSPSerial) upload(ms *Mission, eeprom bool) {
	mlen := len(ms.MissionItems)
	fmt.Printf("upload %d, save %v\n", mlen, eeprom)
	for i, v := range ms.MissionItems {
		fmt.Printf("Upload %d\r", i)
		_, b := serialise_wp(v, (i == mlen-1))
		m.Send_msp(msp_SET_WP, b)
		_, _, err := m.Read_msp()
		if err != nil {
			fmt.Printf("for wp %d, %s\n", i, err)
		}
	}

	if eeprom {
		z := make([]byte, 1)
		z[0] = 1
		m.Send_msp(msp_WP_MISSION_SAVE, z)
		_, _, err := m.Read_msp()
		if err != nil {
			fmt.Printf("failed to save mission: %s\n", err)
		} else {
			fmt.Printf("Saved mission\n")
		}
	}
	m.Send_msp(msp_WP_GETINFO, nil)
	_, payload, err := m.Read_msp()
	if err != nil {
		fmt.Println("read: ", err)
	} else {
		wp_max := payload[1]
		wp_valid := payload[2]
		wp_count := payload[3]
		fmt.Printf("Waypoints: %d of %d, valid %d\n", wp_count, wp_max, wp_valid)
	}
}
