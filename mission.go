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
	"github.com/antchfx/xmlquery"
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

func read_KML(dat []byte) *Mission {
	src := ""
	doc, err := xmlquery.Parse(strings.NewReader(string(dat)))
  if err == nil {
    coords := xmlquery.FindOne(doc, "//Placemark/LineString/coordinates")
    if coords == nil {
      coords = xmlquery.FindOne(doc, "//kml:Placemark/kml:LineString/kml:coordinates")
    }
    if coords != nil {
      src = coords.InnerText()
    }
	}

	items := []MissionItem{}
	version := Version{Value: "wpconv 0.1"}
	mission := &Mission{Version: version, MissionItems: items}

	if src != "" {
		st := strings.Trim(src, "\n\r\t ")
		ss := strings.Split(st, " ")
		n := 1
		for _, val := range ss {
			coords := strings.Split(val, ",")
			if len(coords) > 1 {
				for i, c := range coords {
					coords[i] = strings.Trim(c, "\n\r\t ")
				}

				alt := 0.0
				lon, _ := strconv.ParseFloat(coords[0], 64)
				lat, _ := strconv.ParseFloat(coords[1], 64)
				if len(coords) > 2 {
					alt, _ = strconv.ParseFloat(coords[2], 64)
				}
				item := MissionItem{No: n, Lat: lat, Lon: lon, Alt: int32(alt), Action: "WAYPOINT"}
				n++
				mission.MissionItems = append(mission.MissionItems, item)
			}
		}
	}
	return mission
}

func read_GPX(dat []byte) *Mission {
	items := []MissionItem{}
	version := Version{Value: "wpconv 0.1"}
	mission := &Mission{Version: version, MissionItems: items}

	doc, err := xmlquery.Parse(strings.NewReader(string(dat)))
	stypes := []string{"//trkpt", "//rtept", "//wpt"}

	if err == nil {
		for _,stype:= range stypes {
			list  := xmlquery.Find(doc, stype)
			if list != nil {
				for k, node := range list {
					alt := 0.0
					lat,_ := strconv.ParseFloat(node.SelectAttr("lat"), 64)
					lon,_ := strconv.ParseFloat(node.SelectAttr("lon"), 64)
					enode := xmlquery.FindOne(node, "ele")
					if enode != nil {
						alt,_ = strconv.ParseFloat(enode.InnerText(), 64)
					}
					item := MissionItem{No: k + 1, Lat: lat, Lon: lon, Alt: int32(alt), Action: "WAYPOINT"}
					mission.MissionItems = append(mission.MissionItems, item)
				}
				break
			}
		}
	}
	return mission
}

func (m *Mission) Add_rtl(land bool) {
	k := len(m.MissionItems)
	p1 := int16(0)
	if land {
		p1 = 1
	}
	item := MissionItem{No: k + 1, Lat: 0.0, Lon: 0.0, Alt: 0, Action: "RTH", P1: p1}
	m.MissionItems = append(m.MissionItems, item)
}

func (m *Mission) Dump(path string) {
	s, err := xml.MarshalIndent(m, "", "  ")
	w, err := openStdoutOrFile(path)
	if err == nil {
		defer w.Close()
		//		w.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"))
		w.Write([]byte(xml.Header))
		fmt.Fprintf(w, "%s\n", string(s))
	}
}

func read_Simple(dat []byte) *Mission {
	r := csv.NewReader(strings.NewReader(string(dat)))

	items := []MissionItem{}
	version := Version{Value: "wpconv 0.1"}

	mission := &Mission{Version: version, MissionItems: items}

	n := 1
	has_no := false

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if record[0] == "no" {
			has_no = true
			continue
		}

		if record[0] == "wp" {
			continue
		}

		var lat, lon float64

		j := 0
		no := n
		if has_no {
			no, _ = strconv.Atoi(record[0])
			j = 1
		}

		alt, _ := strconv.ParseFloat(record[j+3], 64)
		fp1, _ := strconv.ParseFloat(record[j+4], 64)
		p1 := int16(0)
		action := record[j]
		switch action {
		case "RTH":
			lat = 0.0
			lon = 0.0
			alt = 0
			if fp1 != 0 {
				p1 = 1
			}
		case "WAYPOINT", "WP":
			action = "WAYPOINT"
			lat, _ = strconv.ParseFloat(record[j+1], 64)
			lon, _ = strconv.ParseFloat(record[j+2], 64)
			if fp1 > 0 {
				p1 = int16(fp1 * 100)
			}
		default:
			continue
		}
		item := MissionItem{No: no, Lat: lat, Lon: lon, Alt: int32(alt), Action: action, P1: p1}
		mission.MissionItems = append(mission.MissionItems, item)
		n++
	}
	return mission
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
		return mtype, nil, err
	} else {
		var m *Mission
		switch {
		case bytes.HasPrefix(dat, []byte("<?xml")):
			switch {
			case bytes.Contains(dat, []byte("MISSIONITEM")):
				m = read_XML_mission(dat)
				mtype = "mwx"
			case bytes.Contains(dat, []byte("<gpx ")):
				m = read_GPX(dat)
				mtype = "gpx"
			case bytes.Contains(dat, []byte("<kml ")):
				m = read_KML(dat)
				mtype = "kml"
			default:
				m = nil
			}
		case bytes.HasPrefix(dat, []byte("QGC WPL")):
			m = read_QML(dat)
			mtype = "qml"
		case bytes.HasPrefix(dat, []byte("no,wp,lat,lon,alt,p1")),
			bytes.HasPrefix(dat, []byte("wp,lat,lon,alt,p1")):
			m = read_Simple(dat)
			mtype = "csv"
		default:
			m = nil
		}
		return mtype, m, nil
	}
}
