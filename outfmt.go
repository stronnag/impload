package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
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

func (mm *MultiMission) Update_mission_meta() {
	var moving = os.Getenv("MWP_POS_OFFSET")
	var offlat, offlon float64
	if moving != "" {
		offsets := strings.Split(moving, ",")
		offlat, _ = strconv.ParseFloat(offsets[0], 64)
		offlon, _ = strconv.ParseFloat(offsets[1], 64)
	}

	ino := 1
	for i := range mm.Segment {
		if *outfmt != "xml-ugly" {
			ino = 1
		}
		var bbox = BBox{-999, 999, -999, 999}
		var cx, cy, ni float64
		for j := range mm.Segment[i].MissionItems {
			mm.Segment[i].MissionItems[j].No = ino
			ino++

			if mm.Segment[i].MissionItems[j].is_GeoPoint() {
				if mm.Segment[i].MissionItems[j].Lat == 0 &&
					mm.Segment[i].MissionItems[j].Lon == 0 {
					mm.Segment[i].MissionItems[j].Flag = 0x48
				}

				if mm.Segment[i].MissionItems[j].Flag == 0x48 {
					if mm.Segment[i].MissionItems[j].Lat == 0 {
						mm.Segment[i].MissionItems[j].Lat = mm.Segment[i].Metadata.Homey
					}
					if mm.Segment[i].MissionItems[j].Lon == 0 {
						mm.Segment[i].MissionItems[j].Lon = mm.Segment[i].Metadata.Homex
					}
				}

				if moving != "" {
					mm.Segment[i].MissionItems[j].Lat += offlat
					mm.Segment[i].MissionItems[j].Lon += offlon
				}
				cy += mm.Segment[i].MissionItems[j].Lat
				cx += mm.Segment[i].MissionItems[j].Lon
				ni++
				if mm.Segment[i].MissionItems[j].Lat > bbox.lamax {
					bbox.lamax = mm.Segment[i].MissionItems[j].Lat
				}
				if mm.Segment[i].MissionItems[j].Lat < bbox.lamin {
					bbox.lamin = mm.Segment[i].MissionItems[j].Lat
				}
				if mm.Segment[i].MissionItems[j].Lon > bbox.lomax {
					bbox.lomax = mm.Segment[i].MissionItems[j].Lon
				}
				if mm.Segment[i].MissionItems[j].Lon < bbox.lomin {
					bbox.lomin = mm.Segment[i].MissionItems[j].Lon
				}
			}
		}
		if ni > 0 {
			mm.Segment[i].Metadata.Cx = cx / ni
			mm.Segment[i].Metadata.Cy = cy / ni
		}
		mm.Segment[i].Metadata.Zoom = evince_zoom(bbox)
		mm.Segment[i].Metadata.Generator = "impload"
		mm.Segment[i].Metadata.Stamp = time.Now().Format(time.RFC3339)
	}
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

func (mm *MultiMission) To_xml(params ...string) {
	mm.Comment = xml_comment(params)
	mm.Update_mission_meta()
	w, err := openStdoutOrFile(params[0])
	if err == nil {
		xs, _ := xml.MarshalIndent(mm, "", " ")
		fmt.Fprint(w, xml.Header)
		fmt.Fprintln(w, string(xs))
	}
}

func (mm *MultiMission) To_json(fname string) {
	w, err := openStdoutOrFile(fname)
	if err == nil {
		defer w.Close()
		mm.Update_mission_meta()
		js, _ := json.Marshal(mm)
		fmt.Fprintln(w, string(js))
	}
}

func (mm *MultiMission) To_cli(fname string) {
	w, err := openStdoutOrFile(fname)
	if err == nil {
		defer w.Close()
		fmt.Fprintln(w, "# wp load")
		nmi := 0
		for _, m := range mm.Segment {
			nmi += len(m.MissionItems)
		}

		fmt.Fprintf(w, "#wp %d valid\n", nmi)
		no := 1
		for _, m := range mm.Segment {
			for _, mi := range m.MissionItems {
				ilat := int(mi.Lat * 1e7)
				ilon := int(mi.Lon * 1e7)
				ialt := int(mi.Alt * 100)
				iact := Encode_action(mi.Action)
				if iact == 6 {
					mi.P1--
				}
				fmt.Fprintf(w, "wp %d %d %d %d %d %d %d %d %d\n",
					no, iact, ilat, ilon, ialt, mi.P1, mi.P2, mi.P3, mi.Flag)
				no++
			}
		}
	}
}

func (mm *MultiMission) To_md(params ...string) {
	w, err := openStdoutOrFile(params[0])
	if err == nil {
		defer w.Close()

		mm.Comment = xml_comment(params)

		fmt.Fprintln(w, "## Mission Details")

		for j, m := range mm.Segment {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "### Segment %d\n", j+1)
			fmt.Fprintln(w)
			fmt.Fprintln(w, "|      |              |")
			fmt.Fprintln(w, "| ---- | ------------ |")
			fmt.Fprintf(w, "| Generator | %s |\n", m.Metadata.Generator)
			fmt.Fprintf(w, "| Save date | %s |\n", m.Metadata.Stamp)
			if m.Metadata.Homey != 0 && m.Metadata.Homex != 0 {
				fmt.Fprintf(w, "| Planned Home | %.7f %.7f |\n", m.Metadata.Homey, m.Metadata.Homex)
			}
			if m.Metadata.Cy != 0 && m.Metadata.Cx != 0 {
				fmt.Fprintf(w, "| Centre on | %.7f %.7f |\n", m.Metadata.Cy, m.Metadata.Cx)
			}

			fmt.Fprintln(w)
			fmt.Fprintln(w, "| WP# | Action | Lat | Lon | Alt | P1 | P2 | P3 | flag |")
			fmt.Fprintln(w, "| ---- | ------ | ---- | ---- | ---- | ---- | ---- | ---- | ---- |")

			no := 1
			for _, mi := range m.MissionItems {
				fmt.Fprintf(w, "| %d | %s | %.7f | %.7f | %d | %d | %d | %d | %d |\n",
					no, mi.Action, mi.Lat, mi.Lon, mi.Alt, mi.P1, mi.P2, mi.P3, mi.Flag)
				no++
			}
		}
		if len(mm.Comment) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, mm.Comment)
		}
	}
}
