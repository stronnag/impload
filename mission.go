package main

import (
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type Version struct {
	Value string `xml:"value,attr"`
}
type MissionItem struct {
	No     int     `xml:"no,attr"`
	Action string  `xml:"action,attr"`
	Lat    float64 `xml:"lat,attr"`
	Lon    float64 `xml:"lon,attr"`
	Alt    int32   `xml:"alt,attr"`
	P1     int16   `xml:"parameter1,attr"`
	P2     uint16  `xml:"parameter2,attr"`
	P3     uint16  `xml:"parameter3,attr"`
}

type Mission struct {
	XMLName      xml.Name      `xml:"MISSION"̀`
	Version      Version       `xml:"VERSION"̀`
	MissionItems []MissionItem `xml:"MISSIONITEM"̀`
}

type WPItem struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Alt  float32   `xml:"ele"`
	Name string  `xml:"name"`
}

type GPX struct {
	XMLName xml.Name `xml:"gpx"̀`
	WPItems []WPItem `xml:"wpt"̀`
}

func read_GPX(dat []byte) (*Mission) {
	var g GPX
	xml.Unmarshal(dat, &g)
	items := []MissionItem{}
	version := Version{Value: "wpconv 0.1"}
	mission := &Mission{Version: version, MissionItems: items}
	for k,v := range g.WPItems {
		item := MissionItem{No: k+1, Lat: v.Lat, Lon: v.Lon, Alt: int32(v.Alt), Action: "WAYPOINT"}
    mission.MissionItems = append(mission.MissionItems, item)
	}
	return mission
}

func (m *Mission) Add_rtl(land bool) {
	k := len(m.MissionItems)
	p1 := int16(0)
	if land {
		p1 = 1
	}
	item := MissionItem{No: k+1, Lat: 0.0, Lon: 0.0, Alt: 0, Action: "RTH", P1: p1}
  m.MissionItems = append(m.MissionItems, item)
}

func (m *Mission) Dump(path string) {
	s, err := xml.MarshalIndent(m, "", "  ")
	w, err := openStdoutOrFile(path)
	if err == nil {
		defer w.Close()
		w.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))
		fmt.Fprintf(w, "%s\n", string(s))
	}
}

func read_QML(dat []byte) *Mission {
	r := csv.NewReader(strings.NewReader(string(dat)))
	r.Comma = '\t'
	r.FieldsPerRecord = -1

	items := []MissionItem{}
	version := Version{Value: "wpconv 0.1"}

	mission := &Mission{Version: version, MissionItems: items}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if len(record) == 12 && record[2] == "3" {
			var lat, lon float64
			var action string
			no, _ := strconv.Atoi(record[0])
			alt, _ := strconv.Atoi(record[10])
			var p1 int16

			if record[3] == "20" {
				lat = 0.0
				lon = 0.0
				if alt == 0 {
					p1 = 1
				}
				alt = 0
				action = "RTH"
			} else {
				p1 = 0
				action = "WAYPOINT"
				lat, _ = strconv.ParseFloat(record[8], 64)
				lon, _ = strconv.ParseFloat(record[9], 64)
			}
			item := MissionItem{No: no, Lat: lat, Lon: lon, Alt: int32(alt), Action: action, P1: p1}
			mission.MissionItems = append(mission.MissionItems, item)
		}
	}
	return mission
}

func read_XML_mission(dat []byte) *Mission {
	var mission Mission
	xml.Unmarshal(dat, &mission)
	return &mission
}

func Read_Mission_File(path string) (string, *Mission, error) {
	var dat []byte
	mtype := ""
	r, err := openStdinOrFile(path)
	if err == nil {
		defer r.Close()
		dat, err = ioutil.ReadAll(r)
	}
	if err != nil {
		return mtype,nil, err
	} else {
		var m *Mission
		switch {
		case bytes.HasPrefix(dat, []byte("<?xml")):
			switch {
			case bytes.Contains(dat, []byte("MISSIONITEM")):
				m = read_XML_mission(dat)
				mtype = "mwx"
			case bytes.Contains(dat, []byte("<wpt lat=")):
				m = read_GPX(dat)
				mtype = "gpx"
			default:
				m = nil
			}
		case bytes.HasPrefix(dat, []byte("QGC WPL")):
			m = read_QML(dat)
			mtype = "qml"
		default:
			m = nil
		}
		return mtype, m, nil
	}
}
