impload - iNav Mission Plan uploader
====================================

# Introduction

[impload](https://github.com/stronnag/impload) is a cross-platform command line application to upload missions in a number of formats to an iNav flight controller. "Alien" formats may also be converted to MW-XML. Supported formats include:

* [MW XML](https://github.com/iNavFlight/inav/tree/master/docs/development/wp_mission_schema) mission files (as used by [mwp](https://github.com/stronnag/mwptools), [inav configurator](https://github.com/iNavFlight/inav-configurator), ezgui, mission planner for inav)
* apmplanner / qgroundcontrol mission files (qgc plan mission and survey at least).
* GPX files (tracks, routes, waypoints)
* KML, KMZ files
* Plain, simple CSV files
* [mwp JSON](https://github.com/stronnag/mwptools/blob/master/samples/mission-schema.json) mission files]
* inav cli `wp` stanzas

Serial devices and TCP are supported for upload / download to / from flight controllers.

Please see the [user guide](https://stronnag.github.io/impload/) for more information.

[YouTube Tutorial](https://www.youtube.com/watch?v=Mktmk_Y6PhM)

```
$ impload --help
Usage of impload [options] command [files ...]
Options:
  -a int
    	Default altitude (m) (default 20)
  -b int
    	Baud rate (default 115200)
  -d string
    	Serial Device
  -fmt string
    	Output format (xml, json, cli, xml-ugly) (default "xml")
  -force-land
    	Adds RTH / Land for 'external' formats
  -force-rth
    	Adds RTH for 'external' formats
  -s float
    	Default speed (m/s)
  -v	Shows version
  command:
	Action required (upload|download|store|restore|convert|test|clear|erase)
```

## Device Name

impload supports the mwp device naming scheme:

* `serial_device[@baudrate]`
* `tcp://host:port`
* `udp://remotehost:remote_port`
* `udp://local_host:local_port/remote_host:remote_port`
* `xx:xx:xx:xx:xx:xx` (raw BT socket, Linux only)

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
30:14:12:02:16:64
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

## Command summary

* upload : upload mission to FC volatile memory
* store : upload mission to FC volatile memory and stores in EEPROM
* download : downloads mission from FC volatile memory
* restore : restores mission from EEROM to FC volatile memory and downloads the mission
* convert : converts alien formats to MW-XML (default), or the format defined by the `-fmt` option.
* test : tests communications with FC
* clear : clear mission in volatile RAM (specifically, uploads a mission with just a single RTH WP, which is always safe).
* erase : erases mission in EEPROM and clears mission in volatile RAM (specifically, uploads and stores a mission with just a single RTH WP, which is always safe).
* multi, multi=n : inav 4.0+; gets / sets the `nav_wp_multi_mission_index` value.

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
