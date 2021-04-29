package main

import (
	"fmt"
	"encoding/json"
	"time"
	"math"
)

type MissionMeta struct {
	Cx        float64   `json:"cx"`
	Cy        float64   `json:"cy"`
	Generator string    `json:"generator"`
	Zoom      int       `json:"zoom"`
	Stamp     time.Time `json:"save-date"`
}


type MetaMission struct {
	Meta         MissionMeta   `json:"meta"`
	MissionItems []MissionItem `json:"mission"`
}

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


func (m *Mission) Get_mission_meta() MissionMeta {
	var bbox = BBox{-999, 999, -999, 999}
	mt := MissionMeta{}
	var cx, cy, ni float64
	for _, mi := range m.MissionItems {
		if mi.is_GeoPoint() {
			cy += mi.Lat
			cx += mi.Lon
			ni += 1
			if mi.Lat > bbox.lamax {
				bbox.lamax = mi.Lat
			}
			if mi.Lat < bbox.lamin {
				bbox.lamin = mi.Lat
			}
			if mi.Lon > bbox.lomax {
				bbox.lomax = mi.Lon
			}
			if mi.Lon < bbox.lomin {
				bbox.lomin = mi.Lon
			}
		}
	}
	if ni > 0 {
		mt.Cx = cx / ni
		mt.Cy = cy / ni
	}
	mt.Zoom = evince_zoom(bbox)
	mt.Generator = "impload"
	mt.Stamp = time.Now()
	return mt
}

func (m *Mission) To_json(fname string) {
	w, err := openStdoutOrFile(fname)
	if err == nil {
		defer w.Close()
		md := MetaMission{}
		md.MissionItems = m.MissionItems
		md.Meta = m.Get_mission_meta()
		js, _ := json.Marshal(md)
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

		for _, mi := range m.MissionItems {
			no := mi.No - 1
			flg := 0
			if mi.No == nmi {
				flg = 0xa5
			}
			ilat := int(mi.Lat * 1e7)
			ilon := int(mi.Lon * 1e7)
			ialt := int(mi.Alt * 100)
			iact := Encode_action(mi.Action)
			if iact == 6 {
				mi.P1 -= 1
			}
			fmt.Fprintf(w, "wp %d %d %d %d %d %d %d %d %d\n",
				no, iact, ilat, ilon, ialt, mi.P1, mi.P2, mi.P3, flg)
		}
	}
}
