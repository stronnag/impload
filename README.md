impload - iNav Mission Plan uploader
====================================

# Introduction

impload is a cross-platform command line application to upload missions in a number of formats to an iNav flight controller. Supported formats include:

* MW XML mission files (as used by [mwp](https://github.com/stronnag/mwptools), ezgui, mission planner for inav)
* apmplanner / qgroundcontrol mission files
* GPX files (tracks, routes, waypoints)
* KML files
* Plain, simple CSV files

Please see the [wiki user guide](https://github.com/stronnag/impload/wiki/impload-User-Guide) for more information and CSV format

```
$ ./impload --help
Usage of ./impload [options] command [files ...]
  -a int
    	Default altitude (m) (default 20)
  -b int
    	Baud rate (default 115200)
  -d string
    	Serial Device
  -force-land
    	Adds RTH / Land for 'external' formats
  -force-rth
    	Adds RTH for 'external' formats
  -s int
    	Default speed (m/s)
  command
	Action required (upload|download|store|restore|convert|test)
```

# Use Cases

* Plan missions in apm / qpc, upload (& save) to iNav
* Plan missions an any GPX creating GIS tool
* Plan mission in Google Earth, save as KML path, upload to the FC
* Backup missions made in the configurator (as MW XML mission files)

# Install

Binaries in the Release area (linux ia32/x86_64/arm7, Win32) if you don't want it build it locally.

From source: `go get github.com/stronnag/impload`, binaries endup in `go/bin`, source in `go/src/github.com/stronnag/impload`. Requires `go` and `git`.

Note: If you clone the repo, then you need to `go get` the dependencies:

```
go get github.com/tarm/serial
go get github.com/beevik/etree
```

# Examples

```
# Linux, detect serial device, test communications
# Linux tries /dev/ttyACM0 and /dev/ttyUSB0 (in that order)
$ ./impload test
2018/05/24 18:08:11 Using device /dev/ttyUSB0 115200
INAV v2.0.0 SPRACINGF3 (e7ca7944) API 2.2 "vtail"
Waypoints: 0 of 60, valid 0

# Linux, detect serial device and upload a qpc /apm mission file
$ ./impload upload samples/qpc_0.txt
2018/05/24 18:09:10 Using device /dev/ttyUSB0 115200
INAV v2.0.0 SPRACINGF3 (e7ca7944) API 2.2 "vtail"
Waypoints: 0 of 60, valid 0
upload 12, save false
Waypoints: 12 of 60, valid 1

#
# Upload / store a GPX file
$ ./impload store samples/qpc_1_trk.gpx
2018/05/24 18:34:49 Using device /dev/ttyUSB0 115200
INAV v2.0.0 SPRACINGF3 (e7ca7944) API 2.2 "vtail"
Waypoints: 11 of 60, valid 1
upload 11, save true
Saved mission
Waypoints: 11 of 60, valid 1

```

```
> REM  Windows, needs a named device to be given
> impload -d COM17 upload samples/google-earth-mission.kml
```

# Postscript

The author knows how to spell "implode".
