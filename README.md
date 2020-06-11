impload - iNav Mission Plan uploader
====================================

# Introduction

[impload](https://github.com/stronnag/impload) is a cross-platform command line application to upload missions in a number of formats to an iNav flight controller. "Alien" formats may also be converted to MW-XML. Supported formats include:

* [MW XML](https://github.com/stronnag/mwptools/blob/master/samples/mw-mission.xsd) mission files (as used by [mwp](https://github.com/stronnag/mwptools), ezgui, mission planner for inav)
* apmplanner / qgroundcontrol mission files
* GPX files (tracks, routes, waypoints)
* KML, KMZ files
* Plain, simple CSV files
* [mwp JSON](https://github.com/stronnag/mwptools/blob/master/samples/mission-schema.json) mission files

Serial devices and TCP are supported

Please see the [wiki user guide](https://github.com/stronnag/impload/wiki/impload-User-Guide) for more information and CSV format

```
$ ./impload --help
Usage of ./impload [options] command [files ...]
  -a int
    	Default altitude (m) (default 20)
  -b int
    	Baud rate (default 115200)
  -d string
    	Device name
  -force-land
    	Adds RTH / Land for 'external' formats
  -force-rth
    	Adds RTH for 'external' formats
  -s int
    	Default speed (m/s)
  command
	Action required (upload|download|store|restore|convert|test)
```

## Device Name

impload supports a subset of the mwp device naming scheme:

* `serial_device[@baudrate]`
* `tcp://host:port`
* `udp://remotehost:remote_port`
* `udp://local_host:local_port/remote_host:remote_port`

The baud rate given as an extended device name is preferred to -b

For ESP8288 transparent serial over UDP (the recommended mode for ESP8266), the latter form is needed.

### Device name examples:

```
/dev/ttyUSB0@57600
/dev/ttyACM0
COM17@115200
tcp://esp8266:23
udp://:14014/esp-air:14014
# both sides use port 14014, remote (FC) is esp-air, blank local name is understood as INADDR_ANY.
```

## Use Cases

* Plan missions in apmplanner2 (QGC WPL 110 text files), upload (& save) to iNav
* Plan missions in qgroundcontrol (JSON plan files), upload (& save) to iNav
* Plan missions an any GPX creating GIS tool
* Plan mission in Google Earth, save as KML path, upload to the FC
* Convert "alien" formats to MW-XML.

## Install

Binaries in the Release area (linux ia32/x86_64/arm7, FreeBSD, MacOS, Win32) if you don't want it build it locally.

From source: `go get github.com/stronnag/impload`, binaries endup in `go/bin`, source in `go/src/github.com/stronnag/impload`. Requires `go` and `git`.

## Examples

```
# Linux, detect serial device, test communications
# Linux tries /dev/ttyACM0 and /dev/ttyUSB0 (in that order)
$ ./impload test
2018/05/24 18:08:11 Using device /dev/ttyUSB0 115200
INAV v2.0.0 SPRACINGF3 (e7ca7944) API 2.2 "vtail"
Waypoints: 0 of 60, valid 0

# Linux, detect serial device and upload a apmplanner2 mission file
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

# TCP ...
#
$ impload  -d tcp://localhost:4321 upload samples/qpc_1.mission
2018/09/18 18:57:08 Using device localhost 4321
INAV v2.1.0 SPRACINGF3 (a29bfbd1) API 2.2
Waypoints: 0 of 60, valid 0
upload 12, save false
Waypoints: 12 of 60, valid 1

$ impload  -d tcp://localhost:4321 download /tmp/m.mission
2018/09/18 19:08:30 Using device localhost 4321
INAV v2.1.0 SPRACINGF3 (a29bfbd1) API 2.2
Waypoints: 12 of 60, valid 1

# UDP ....
#
$ impload -d udp://:14014/esp-air:14014 test
2018/09/19 21:11:34 Using device udp://:14014/esp-air:14014
INAV v2.1.0 SPRACINGF3 (a29bfbd1) API 2.2
Waypoints: 12 of 60, valid 1

$ impload -d udp://:14014/esp-air:14014 download /tmp/m.mission
2018/09/19 21:11:46 Using device udp://:14014/esp-air:14014
INAV v2.1.0 SPRACINGF3 (a29bfbd1) API 2.2
Waypoints: 12 of 60, valid 1

> REM  Windows, needs a named device to be given
> impload -d COM17 upload samples/google-earth-mission.kml

# Conversion
$ impload convert g-earth.kmz example.mission
```

## Postscript

The author knows how to spell "implode".
