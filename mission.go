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
	"os"
	"github.com/beevik/etree"
	"archive/zip"
	"encoding/json"
	"path"
)

type QGCrec struct {
	jindex  int
	command int
	lat     float64
	lon     float64
	alt     float64
	params  [4]float64
}

type MissionItem struct {
	No     int     `json:"no"`
	Action string  `json:"action"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Alt    int32   `json:"alt"`
	P1     int16   `json:"p1"`
	P2     int16   `json:"p2"`
	P3     uint16  `json:"p3"`
}

type Mission struct {
	Version string
	MissionItems []MissionItem
}

func read_kml(dat []byte) *Mission {

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
					/*
						if len(coords) > 2 {
							alt, _ = strconv.ParseFloat(coords[2], 64)
						}
					*/
					item := MissionItem{No: n, Lat: lat, Lon: lon, Alt: int32(alt), Action: "WAYPOINT"}
					n++
					mission.MissionItems = append(mission.MissionItems, item)
				}
			}
		}
	}
	return mission
}

func read_gpx(dat []byte) *Mission {
	items := []MissionItem{}
	mission := &Mission{GetVersion(), items}
	stypes := []string{"//trkpt", "//rtept", "//wpt"}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(dat); err == nil {
		root := doc.SelectElement("gpx")
		for _, stype := range stypes {
			for k, pts := range root.FindElements(stype) {
				alt := 0.0
				lat, _ := strconv.ParseFloat(pts.SelectAttrValue("lat", "0"), 64)
				lon, _ := strconv.ParseFloat(pts.SelectAttrValue("lon", "0"), 64)
				/*
									if anode := pts.SelectElement("ele"); anode != nil {
										alt,_ = strconv.ParseFloat(anode.Text(), 64)
					        }
				*/
				item := MissionItem{No: k + 1, Lat: lat, Lon: lon, Alt: int32(alt), Action: "WAYPOINT"}
				mission.MissionItems = append(mission.MissionItems, item)
			}
			break
		}
	}
	return mission
}

func (m *Mission) is_valid() bool {
	force := os.Getenv("IMPLOAD_NO_VERIFY")
	if len(force) > 0 {
		return true
	}
	mlen := int16(len(m.MissionItems))
	if mlen > 60 {
		return false
	}
	// Urg, Urg array index v. WP Nos ......
	for i := int16(0); i < mlen; i++ {
		var target = m.MissionItems[i].P1 - 1
		if m.MissionItems[i].Action == "JUMP" {
			if (i == 0) || ((target > (i - 2)) && (target < (i + 2))) || (target >= mlen) || (m.MissionItems[i].P2 < -1) {
				return false
			}
			if !(m.MissionItems[target].Action == "WAYPOINT" || m.MissionItems[target].Action == "POSHOLD_TIME" || m.MissionItems[target].Action == "LAND") {
				return false
			}
		}
	}
	return true
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

func (m *Mission) Dump(outfmt string, params ...string) {
	switch outfmt {
	case "cli":
		m.To_cli(params[0])
	case "json":
		m.To_json(params[0])
	default:
		m.To_xml(params...)
	}
}

func (m *Mission) To_xml(params ...string) {
	t := time.Now()
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="utf-8"`)
	x := doc.CreateElement("mission")

	var sb strings.Builder
	sb.WriteString("Created by \"impload\" ")
	sb.WriteString(GitTag)
	sb.WriteString(" on ")
	sb.WriteString(t.Format(time.RFC3339))
	if len(params) > 1 {
		sb.WriteString("\n      from ")
		if params[1] == "-" {
			sb.WriteString("<stdin>")
		} else {
			sb.WriteString(path.Base(params[1]))
		}
		if len(params) > 2 {
			sb.WriteString(" (")
			sb.WriteString(params[2])
			sb.WriteByte(')')
		}
	}
	sb.WriteString("\n      <https://github.com/stronnag/impload>\n")
	x.CreateComment(sb.String())
	v := x.CreateElement("version")
	v.CreateAttr("value", m.Version)
	for _, mi := range m.MissionItems {
		xi := x.CreateElement("missionitem")
		xi.CreateAttr("no", fmt.Sprintf("%d", mi.No))
		xi.CreateAttr("action", mi.Action)
		xi.CreateAttr("lat", strconv.FormatFloat(mi.Lat, 'f', 7, 64))
		xi.CreateAttr("lon", strconv.FormatFloat(mi.Lon, 'f', 7, 64))
		xi.CreateAttr("alt", fmt.Sprintf("%d", mi.Alt))
		xi.CreateAttr("parameter1", fmt.Sprintf("%d", mi.P1))
		xi.CreateAttr("parameter2", fmt.Sprintf("%d", mi.P2))
		xi.CreateAttr("parameter3", fmt.Sprintf("%d", mi.P3))
	}
	w, err := openStdoutOrFile(params[0])
	if err == nil {
		doc.Indent(2)
		doc.WriteTo(w)
	}
}

func read_simple(dat []byte) *Mission {
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

		p1 := int16(0)
		p2 := int16(0)
		fp2 := 0.0
		lat, _ = strconv.ParseFloat(record[j+1], 64)
		lon, _ = strconv.ParseFloat(record[j+2], 64)
		alt, _ := strconv.ParseFloat(record[j+3], 64)
		fp1, _ := strconv.ParseFloat(record[j+4], 64)
		if len(record) > j+5 {
			fp2, _ = strconv.ParseFloat(record[j+5], 64)
		}
		var action string

		iaction, err := strconv.Atoi(record[j])
		if err == nil {
			action = Decode_action(byte(iaction))
		} else {
			action = record[j]
		}
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
			if fp1 > 0 {
				p1 = int16(fp1 * 100)
			}
		case "POSHOLD_TIME":
			if fp2 > 0 {
				p2 = int16(fp2 * 100)
			}
			p1 = int16(fp1)
		case "JUMP":
			lat = 0.0
			lon = 0.0
			p1 = int16(fp1)
			p2 = int16(fp2)
		case "LAND":
			if fp1 > 0 {
				p1 = int16(fp1 * 100)
			}
		case "SET_POI":
		case "SET_HEAD":
			p1 = int16(fp1)
		default:
			continue
		}
		item := MissionItem{No: no, Lat: lat, Lon: lon, Alt: int32(alt), Action: action, P1: p1, P2: p2}
		mission.MissionItems = append(mission.MissionItems, item)
		n++
	}
	return mission
}

func read_qgc_json(dat []byte) []QGCrec {
	qgcs := []QGCrec{}
	var result map[string]interface{}

	json.Unmarshal(dat, &result)
	mi := result["mission"].(interface{})
	mid := mi.(map[string]interface{})
	it := mid["items"].([]interface{})

	for _, l := range it {
		ll := l.(map[string]interface{})
		ps := ll["params"].([]interface{})
		qg := QGCrec{}
		qg.jindex = int(ll["doJumpId"].(float64))
		qg.command = int(ll["command"].(float64))
		qg.lat = ps[4].(float64)
		qg.lon = ps[5].(float64)
		qg.alt = ps[6].(float64)
		for j := 0; j < 4; j++ {
			if ps[j] != nil {
				qg.params[j] = ps[j].(float64)
			}
		}
		qgcs = append(qgcs, qg)
	}
	return qgcs
}

func read_qgc_text(dat []byte) []QGCrec {
	qgcs := []QGCrec{}

	r := csv.NewReader(strings.NewReader(string(dat)))
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err == nil {
		for _, record := range records {
			if len(record) == 12 {
				no, err := strconv.Atoi(record[0])
				if err == nil && no > 0 {
					qg := QGCrec{}
					qg.jindex = no
					qg.command, _ = strconv.Atoi(record[3])
					qg.alt, _ = strconv.ParseFloat(record[10], 64)
					qg.lat, _ = strconv.ParseFloat(record[8], 64)
					qg.lon, _ = strconv.ParseFloat(record[9], 64)
					for j := 0; j < 4; j++ {
						qg.params[j], _ = strconv.ParseFloat(record[4+j], 64)
					}
					qgcs = append(qgcs, qg)
				}
			}
		}
	} else {
		log.Fatal(err)
	}
	return qgcs
}

func fixup_qgc_mission(mission *Mission, have_jump bool) (*Mission, bool) {
	ok := true
	if have_jump {
		for i := 0; i < len(mission.MissionItems); i++ {
			if mission.MissionItems[i].Action == "JUMP" {
				jumptgt := mission.MissionItems[i].P1
				ajump := int16(0)
				for j := 0; j < len(mission.MissionItems); j++ {
					if mission.MissionItems[j].P3 == uint16(jumptgt) {
						ajump = int16(j + 1)
						break
					}
				}
				if ajump == 0 {
					ok = false
				} else {
					mission.MissionItems[i].P1 = ajump
				}
				no := int16(i + 1) // item index
				if mission.MissionItems[i].P1 < 1 || ((mission.MissionItems[i].P1 > no-2) &&
					(mission.MissionItems[i].P1 < no+2)) {
					ok = false
				}
			}
		}
	}
	if ok {
		for i := 0; i < len(mission.MissionItems); i++ {
			mission.MissionItems[i].P3 = 0
		}
		return mission, ok
	} else {
		return nil, false
	}
}

func process_qgc(dat []byte, mtype string) *Mission {
	var qs []QGCrec
	items := []MissionItem{}
	mission := &Mission{GetVersion(), items}

	if mtype == "qgc-text" {
		qs = read_qgc_text(dat)
	} else {
		qs = read_qgc_json(dat)
	}
	last_alt := 0.0
	last_lat := 0.0
	last_lon := 0.0

	have_land := false
	lastj := -1

	for j, rq := range qs {
		if rq.command == 20 {
			lastj = j
		} else if rq.command == 21 && j == lastj+1 {
			have_land = true
		}
	}

	last := false
	have_jump := false

	no := 0
	for _, q := range qs {
		ok := true
		var action string
		var p1, p2 int16

		switch q.command {
		case 16:
			if q.params[0] == 0 {
				action = "WAYPOINT"
				p1 = 0
			} else {
				action = "POSHOLD_TIME"
				p1 = int16(q.params[0])
			}

		case 19:
			action = "POSHOLD_TIME"
			p1 = int16(q.params[0])
			if q.alt == 0 {
				q.alt = last_alt
			}
			if q.lat == 0.0 {
				q.lat = last_lat
			}
			if q.lon == 0.0 {
				q.lon = last_lon
			}
		case 20:
			action = "RTH"
			q.lat = 0.0
			q.lon = 0.0
			if q.alt == 0 || have_land {
				p1 = 1
			}
			q.alt = 0
			last = true

		case 21:
			action = "LAND"
			p1 = 0
			if q.alt == 0 {
				q.alt = last_alt
			}
			if q.lat == 0.0 {
				q.lat = last_lat
			}
			if q.lon == 0.0 {
				q.lon = last_lon
			}
		case 177:
			p1 = int16(q.params[0])
			action = "JUMP"
			p2 = int16(q.params[1])
			q.lat = 0.0
			q.lon = 0.0
			have_jump = true

		case 195, 201:
			action = "SET_POI"

		case 115:
			p1 = int16(q.params[0])
			act := int(q.params[3])
			if p1 == 0 && act == 0 {
				p1 = -1
			}
			action = "SET_HEAD"
			q.lat = 0
			q.lon = 0
			q.alt = 0

		case 197:
			p1 = -1
			action = "SET_HEAD"
			q.lat = 0
			q.lon = 0
			q.alt = 0

		default:
			ok = false
		}
		if ok {
			last_alt = q.alt
			last_lat = q.lat
			last_lon = q.lon
			p3 := uint16(q.jindex)
			no += 1
			item := MissionItem{No: no, Lat: q.lat, Lon: q.lon, Alt: int32(q.alt), Action: action, P1: p1, P2: p2, P3: p3}
			mission.MissionItems = append(mission.MissionItems, item)
			if last {
				break
			}
		}
	}

	mission, ok := fixup_qgc_mission(mission, have_jump)
	if !ok {
		log.Fatalf("Unsupported QGC file\n")
	}
	return mission
}

func read_xml_mission(dat []byte) *Mission {
	items := []MissionItem{}
	mission := &Mission{"impload", items}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(dat); err == nil {
		for _, root := range doc.ChildElements() {
			if strings.EqualFold(root.Tag, "MISSION") {
				for _, el := range root.ChildElements() {
					switch {
					case strings.EqualFold(el.Tag, "VERSION"):
						version := el.SelectAttrValue("value", "")
						if version != "" {
							mission.Version = version
						}
					case strings.EqualFold(el.Tag, "MISSIONITEM"):
						no, _ := strconv.Atoi(el.SelectAttrValue("no", "0"))
						action := el.SelectAttrValue("action", "WAYPOINT")
						lat, _ := strconv.ParseFloat(el.SelectAttrValue("lat", "0"), 64)
						lon, _ := strconv.ParseFloat(el.SelectAttrValue("lon", "0"), 64)
						alt, _ := strconv.Atoi(el.SelectAttrValue("alt", "0"))
						p1, _ := strconv.Atoi(el.SelectAttrValue("parameter1", "0"))
						p2, _ := strconv.Atoi(el.SelectAttrValue("parameter2", "0"))
						p3, _ := strconv.Atoi(el.SelectAttrValue("parameter3", "0"))
						item := MissionItem{no, action, lat, lon, int32(alt), int16(p1), int16(p2), uint16(p3)}
						mission.MissionItems = append(mission.MissionItems, item)
					default:
						// fmt.Printf("ignoring tag %s\n", el.Tag)
					}
				}
			}
		}
	}
	return mission
}

func read_kmz(path string) (string, *Mission) {
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
					return mtype, m
				}
			}
		}
	}
	return "", nil
}

func read_json(dat []byte) *Mission {
	items := []MissionItem{}
	mission := &Mission{"impload", items}
	var result map[string]interface{}
	json.Unmarshal(dat, &result)
	mi := result["mission"].([]interface{})
	for _, l := range mi {
		ll := l.(map[string]interface{})
		item := MissionItem{int(ll["no"].(float64)), ll["action"].(string),
			ll["lat"].(float64), ll["lon"].(float64),
			int32(ll["alt"].(float64)), int16(ll["p1"].(float64)),
			int16(ll["p2"].(float64)), uint16(ll["p3"].(float64))}
		mission.MissionItems = append(mission.MissionItems, item)
	}
	return mission
}

func read_inav_cli(dat []byte) *Mission {
	items := []MissionItem{}
	mission := &Mission{"impload", items}
	for _, ln := range strings.Split(string(dat), "\n") {
		if strings.HasPrefix(ln, "wp ") {
			parts := strings.Split(ln, " ")
			if len(parts) == 10 {
				no, _ := strconv.Atoi(parts[1])
				iact, _ := strconv.Atoi(parts[2])
				ilat, _ := strconv.Atoi(parts[3])
				ilon, _ := strconv.Atoi(parts[4])
				alt, _ := strconv.Atoi(parts[5])
				p1, _ := strconv.Atoi(parts[6])
				p2, _ := strconv.Atoi(parts[7])
				p3, _ := strconv.Atoi(parts[8])
				flg, _ := strconv.Atoi(parts[9])
				lat := float64(ilat) / 1.0e7
				lon := float64(ilon) / 1.0e7
				action := Decode_action(byte(iact))
				if iact == 6 {
					p1 += 1
				}
				no += 1
				alt /= 100
				item := MissionItem{no, action, lat, lon, int32(alt), int16(p1), int16(p2), uint16(p3)}
				mission.MissionItems = append(mission.MissionItems, item)
				if flg == 0xa5 {
					break
				}
			}
		}
	}
	return mission
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
		if !m.is_valid() {
			fmt.Fprintf(os.Stderr, "Note: Mission fails verification\n")
		}
		return mtype, m, nil
	}
}

func handle_mission_data(dat []byte, path string) (string, *Mission) {
	var m *Mission
	mtype := ""
	switch {
	case bytes.HasPrefix(dat, []byte("<?xml")):
		switch {
		case bytes.Contains(dat, []byte("<MISSION")),
			bytes.Contains(dat, []byte("<mission")):
			m = read_xml_mission(dat)
			mtype = "mwx"
		case bytes.Contains(dat, []byte("<gpx ")):
			m = read_gpx(dat)
			mtype = "gpx"
		case bytes.Contains(dat, []byte("<kml ")):
			m = read_kml(dat)
			mtype = "kml"
		default:
			m = nil
		}
	case bytes.HasPrefix(dat, []byte("QGC WPL 110")):
		mtype = "qgc-text"
		m = process_qgc(dat, mtype)
	case bytes.HasPrefix(dat, []byte("no,wp,lat,lon,alt,p1")),
		bytes.HasPrefix(dat, []byte("wp,lat,lon,alt,p1")):
		m = read_simple(dat)
		mtype = "csv"
	case bytes.HasPrefix(dat, []byte("PK\003\004")):
		mtype, m = read_kmz(path)
	case bytes.HasPrefix(dat, []byte("{\"meta\":{")):
		mtype = "mwp-json"
		m = read_json(dat)
	case bytes.Contains(dat[0:100], []byte("\"fileType\": \"Plan\"")):
		mtype = "qgc-json"
		m = process_qgc(dat, mtype)
	case bytes.HasPrefix(dat, []byte("# wp")), bytes.HasPrefix(dat, []byte("#wp")), bytes.HasPrefix(dat, []byte("wp 0")):
		mtype = "inav cli"
		m = read_inav_cli(dat)
	default:
		m = nil
	}
	return mtype, m
}
