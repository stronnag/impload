package main

import (
	"fmt"
	"encoding/json"
	"time"
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

func (mi *MissionItem) is_GeoPoint() bool {
	a := mi.Action
	return !(a == "RTH" || a == "SET_HEAD" || a == "JUMP")
}

func (m *Mission) To_json(fname string) {
	w, err := openStdoutOrFile(fname)
	if err == nil {
		defer w.Close()
		md := MetaMission{}
		var cx, cy, ni float64

		md.MissionItems = m.MissionItems
		for _, mi := range md.MissionItems {
			if mi.is_GeoPoint() {
				cy += mi.Lat
				cx += mi.Lon
				ni += 1
			}
		}

		md.Meta.Cx = cx / ni
		md.Meta.Cy = cy / ni
		md.Meta.Generator = "impload"
		md.Meta.Zoom = 0
		md.Meta.Stamp = time.Now()

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
