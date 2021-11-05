package main

import (
	"fmt"
	"encoding/json"
	"encoding/xml"
	"time"
	"math"
	"path"
	"strings"
	"os"
	"strconv"
)

type BBox struct {
	lamax, lamin, lomax, lomin float64
}

func evince_zoom(bbox BBox) int {
	alat := (bbox.lamax + bbox.lamin) / 2
	vrng := (bbox.lamax - bbox.lamin)
	hrng := (bbox.lomax - bbox.lomin) / math.Cos(alat*math.Pi/180.0) // well, sort of
	drng := math.Sqrt(vrng*vrng+hrng*hrng) * 60 * 1852 * 0.7
	//	fmt.Printf("v %.0f, h %.0f, d %.0f\n", vrng*60*1852, hrng*60*1852, drng)
	z := 0
	switch {
	case drng < 120:
		z = 20
	case drng < 240:
		z = 19
	case drng < 480:
		z = 18
	case drng < 960:
		z = 17
	case drng < 1920:
		z = 16
	case drng < 1920:
		z = 15
	case drng < 3840:
		z = 14
	case drng < 7680:
		z = 14
	}
	return z
}

func (m *Mission) Update_mission_meta() {
	var bbox = BBox{-999, 999, -999, 999}
	var cx, cy, ni float64
	var offlat, offlon float64
	var moving = os.Getenv("MWP_POS_OFFSET")
	if moving != "" {
		offsets := strings.Split(moving, ",")
		offlat, _ = strconv.ParseFloat(offsets[0], 64)
		offlon, _ = strconv.ParseFloat(offsets[1], 64)
	}

	ino := 1
	for j := range m.MissionItems {
		m.MissionItems[j].No = ino
		if m.MissionItems[j].Flag == 0xa5 {
			ino = 1
		} else {
			ino++
		}

		if m.MissionItems[j].is_GeoPoint() {
			if moving != "" {
				m.MissionItems[j].Lat += offlat
				m.MissionItems[j].Lon += offlon
			}
			cy += m.MissionItems[j].Lat
			cx += m.MissionItems[j].Lon
			ni++
			if m.MissionItems[j].Lat > bbox.lamax {
				bbox.lamax = m.MissionItems[j].Lat
			}
			if m.MissionItems[j].Lat < bbox.lamin {
				bbox.lamin = m.MissionItems[j].Lat
			}
			if m.MissionItems[j].Lon > bbox.lomax {
				bbox.lomax = m.MissionItems[j].Lon
			}
			if m.MissionItems[j].Lon < bbox.lomin {
				bbox.lomin = m.MissionItems[j].Lon
			}
		}
	}
	if ni > 0 {
		m.Metadata.Cx = cx / ni
		m.Metadata.Cy = cy / ni
	}
	m.Metadata.Zoom = evince_zoom(bbox)
	m.Metadata.Generator = "impload"
	m.Metadata.Stamp = time.Now().Format(time.RFC3339)
}

func xml_comment(params []string) string {
	var sb strings.Builder
	sb.WriteString("Created by \"impload\" ")
	sb.WriteString(GitTag)
	sb.WriteString(" on ")
	sb.WriteString(time.Now().Format(time.RFC3339))
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
	return sb.String()
}

func (m *Mission) To_xml(params ...string) {
	m.Comment = xml_comment(params)
	m.Update_mission_meta()
	w, err := openStdoutOrFile(params[0])
	if err == nil {
		xs, _ := xml.MarshalIndent(m, "", " ")
		fmt.Fprint(w, xml.Header)
		fmt.Fprintln(w, string(xs))
	}
}

func (m *Mission) To_json(fname string) {
	w, err := openStdoutOrFile(fname)
	if err == nil {
		defer w.Close()
		m.Update_mission_meta()
		js, _ := json.Marshal(m)
		fmt.Fprintln(w, string(js))
	}
}

func (m *Mission) To_cli(fname string) {
	w, err := openStdoutOrFile(fname)
	if err == nil {
		defer w.Close()
		nmi := len(m.MissionItems)
		fmt.Fprintln(w, "# wp load")
		fmt.Fprintf(w, "#wp %d valid\n", nmi)

		no := 1
		for _, mi := range m.MissionItems {
			flg := 0
			if no == nmi {
				flg = 0xa5
			}
			ilat := int(mi.Lat * 1e7)
			ilon := int(mi.Lon * 1e7)
			ialt := int(mi.Alt * 100)
			iact := Encode_action(mi.Action)
			if iact == 6 {
				mi.P1--
			}
			fmt.Fprintf(w, "wp %d %d %d %d %d %d %d %d %d\n",
				no, iact, ilat, ilon, ialt, mi.P1, mi.P2, mi.P3, flg)
			no++
		}
	}
}
