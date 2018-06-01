package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"
  "github.com/beevik/etree"
	"archive/zip"
)

type MissionItem struct {
	No     int
	Action string
	Lat    float64
	Lon    float64
	Alt    int32
	P1     int16
	P2     uint16
	P3     uint16
}

type Mission struct {
	Version      string
	MissionItems []MissionItem
}

func read_KML(dat []byte) *Mission {

	items := []MissionItem{}
	mission := &Mission{GetVersion(), items}

  doc := etree.NewDocument()
	if err := doc.ReadFromBytes(dat); err == nil {
		root := doc.SelectElement("kml")
		if src := root.FindElement("//Placemark/LineString/coordinates"); src != nil {
			st := strings.Trim(src.Text(), "\n\r\t ")
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
	}
	return mission
}

func read_GPX(dat []byte) *Mission {
	items := []MissionItem{}
	mission := &Mission{GetVersion(), items}
	stypes := []string{"//trkpt", "//rtept", "//wpt"}

  doc := etree.NewDocument()
  if err := doc.ReadFromBytes(dat); err == nil {
		root := doc.SelectElement("gpx")
		for _,stype:= range stypes {
			for k, pts := range root.FindElements(stype) {
				alt := 0.0
				lat,_ := strconv.ParseFloat(pts.SelectAttrValue("lat","0"), 64)
				lon,_ := strconv.ParseFloat(pts.SelectAttrValue("lon","0"), 64)
				if anode := pts.SelectElement("ele"); anode != nil {
					alt,_ = strconv.ParseFloat(anode.Text(), 64)
        }
				item := MissionItem{No: k + 1, Lat: lat, Lon: lon, Alt: int32(alt), Action: "WAYPOINT"}
				mission.MissionItems = append(mission.MissionItems, item)
			}
			break
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
	t := time.Now()
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)
	x := doc.CreateElement("MISSION")
	x.CreateComment(fmt.Sprintf("Created by \"impload\" v%s on %s\n      <https://github.com/stronnag/impload>\n  ",VERSION,t.Format(time.RFC3339)))
	v := x.CreateElement("VERSION")
	v.CreateAttr("value", m.Version)
	for _,mi := range m.MissionItems {
		xi := x.CreateElement("MISSIONITEM")
		xi.CreateAttr("no", fmt.Sprintf("%d", mi.No))
		xi.CreateAttr("action", mi.Action)
		xi.CreateAttr("lat", strconv.FormatFloat(mi.Lat, 'g', -1, 64))
		xi.CreateAttr("lon", strconv.FormatFloat(mi.Lon, 'g', -1, 64))
		xi.CreateAttr("alt", fmt.Sprintf("%d",mi.Alt))
		xi.CreateAttr("parameter1", fmt.Sprintf("%d", mi.P1))
		xi.CreateAttr("parameter2", fmt.Sprintf("%d", mi.P2))
		xi.CreateAttr("parameter3", fmt.Sprintf("%d", mi.P3))
	}
	w, err := openStdoutOrFile(path)
	if err == nil {
		doc.Indent(2)
		doc.WriteTo(w)
	}
}

func read_Simple(dat []byte) *Mission {
	r := csv.NewReader(strings.NewReader(string(dat)))

	items := []MissionItem{}
	mission := &Mission{GetVersion(), items}

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
	mission := &Mission{GetVersion(), items}

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
	items := []MissionItem{}
	mission := &Mission{"impload", items}
  doc := etree.NewDocument()
  if err := doc.ReadFromBytes(dat); err == nil {
		root := doc.SelectElement("MISSION")
		if vers := root.SelectElement("VERSION"); vers != nil {
			version := vers.SelectAttrValue("value","")
			if version != "" {
				mission.Version = version
			}
		}
		for _, pts := range root.FindElements("//MISSIONITEM") {
			no, _ := strconv.Atoi(pts.SelectAttrValue("no","0"))
			action := pts.SelectAttrValue("action","WAYPOINT")
			lat,_ := strconv.ParseFloat(pts.SelectAttrValue("lat","0"), 64)
			lon,_ := strconv.ParseFloat(pts.SelectAttrValue("lon","0"), 64)
			alt, _ := strconv.Atoi(pts.SelectAttrValue("alt","0"))
			p1, _ := strconv.Atoi(pts.SelectAttrValue("parameter1","0"))
			p2, _ := strconv.Atoi(pts.SelectAttrValue("parameter2","0"))
			p3, _ := strconv.Atoi(pts.SelectAttrValue("parameter3","0"))
			item := MissionItem{no, action, lat, lon, int32(alt), int16(p1), uint16(p2), uint16(p3)}
			mission.MissionItems = append(mission.MissionItems, item)
		}
	}
	return mission
}

func ReadKMZ(path string) (string, *Mission) {
	r, err := zip.OpenReader(path)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		rc, err := f.Open()
		defer rc.Close()
		if err == nil {
			dat, err := ioutil.ReadAll(rc)
			if err == nil {
				mtype, m := handle_mission_data(dat, path)
				if m != nil {
					return mtype,m
				}
			}
		}
	}
	return "",nil
}

func Read_Mission_File(path string) (string, *Mission, error) {
	var dat []byte
	r, err := openStdinOrFile(path)
	if err == nil {
		defer r.Close()
		dat, err = ioutil.ReadAll(r)
	}
	if err != nil {
		return "?", nil, err
	} else {
		mtype, m := handle_mission_data(dat, path)
		return mtype, m, nil
	}
}

func handle_mission_data(dat []byte, path string) (string, *Mission) {
	var m *Mission
	mtype := ""
	switch {
	case bytes.HasPrefix(dat, []byte("<?xml")):
		switch {
		case bytes.Contains(dat, []byte("<MISSION")):
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
	case bytes.HasPrefix(dat,[]byte("PK\003\004")):
		fmt.Printf("KMZ %s\n", path)
		mtype, m = ReadKMZ(path)
	default:
		m = nil
	}
	return mtype,m
}
