impload - iNav Mission Plan uploader
====================================

# Introduction

impload is a cross-platform command line application to upload missions in a number of formats to an iNav flight controller. Supported formats include:

* MW XML mission files (as used by [mwp](https://github.com/stronnag/mwptools), ezgui, mission planner for inav)
* apmplanner / qgroundcontrol mission files
* GPX waypoint files

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
* Backup missions made in the configurator

# Install

From source: `go get github.com/stronnag/impload`

Binaries in the Release area (linux ia32/x86_64/arm7, Win32)

# Examples

```
# Linux, detect serial device, test communications 
$ ./impload test
2018/05/24 18:08:11 Using device /dev/ttyUSB0 115200
INAV v2.0.0 SPRACINGF3 (e7ca7944) API 2.2 "vtail"
Waypoints: 0 of 60, valid 0

# Linux, detect serial device, qpc /apm mission file
$ ./impload upload samples/qpc_0.txt 
2018/05/24 18:09:10 Using device /dev/ttyUSB0 115200
INAV v2.0.0 SPRACINGF3 (e7ca7944) API 2.2 "vtail"
Waypoints: 0 of 60, valid 0
upload 12, save false
Waypoints: 12 of 60, valid 1

#
# Convert a GPX file for tracks (trkpt) to waypoints, and upload  
# Converted GPX output piped into impload
$ gpsbabel -i gpx -f samples/qpc_1_trk.gpx -x transform,wpt=trk -o gpx -F-  | ./impload store -
2018/05/24 18:34:49 Using device /dev/ttyUSB0 115200
INAV v2.0.0 SPRACINGF3 (e7ca7944) API 2.2 "vtail"
Waypoints: 11 of 60, valid 1
upload 11, save true
Saved mission
Waypoints: 11 of 60, valid 1

```

```
> REM  Windows, needs a named device
> impload -d COM17 upload qpc_0.txt
```

