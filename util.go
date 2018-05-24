package main

import (
	"io"
	"os"
)


func openStdinOrFile(path string) (io.ReadCloser, error) {
	var err error
	var r io.ReadCloser

	if len(path) == 0 || path == "-" {
		r = os.Stdin
	} else {
		r, err = os.Open(path)
	}
	return r, err
}

func openStdoutOrFile(path string) (io.WriteCloser, error) {
	var err error
	var w io.WriteCloser

	if len(path) == 0 || path == "-" {
		w = os.Stdout
	} else {
		w, err = os.Create(path)
	}
	return w, err
}
