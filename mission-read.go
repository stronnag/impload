package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func Read_Mission_File(path string) (string, *MultiMission, error) {
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
		if m == nil || !m.is_valid() {
			fmt.Fprintf(os.Stderr, "Note: Mission fails verification %s\n", mtype)
		}
		return mtype, m, nil
	}
}
