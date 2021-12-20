# Conversion between QGC waypoints and inav

| QGC WP 110 Type | QGC numeric value | inav WP |
| --------------- | ----------------- | ------- |
| WAYPOINT | 16 | WAYPOINT |
| WAYPOINT (with hold time) | 16| POSHOLD_TIME |
| Loiter Time | 19 | POSHOLD_TIME |
| Jump to index | 177 | JUMP |
| Return to launch | 20 | RTH |
| Land | 21 | LAND |
| SET_ROI | 201 | SET_POI |
| DO_SET_ROI_LOCATION | 195 | SET_POI |
| DO_SET_ROI_NONE | 197 | SET_HEAD (-1) |
| DO_CONDITION_YAW | 115 | SET_HEAD |

Conversion will fail for any other QGC WP 110 WP types.

Note that the whole SET_POI / SET_HEAD / SET_ROI / DO_SET_ROI_LOCATION / DO_SET_ROI_NONE /  DO_CONDITION_YAW is somewhat problematic.

* SET_ROI / DO_SET_ROI_LOCATION are always mapped to SET_POI regardless of any QGC parameters (which don't seem to be set consistently / if at all between apmplanner and qgroundcontrol.

* DO_SET_ROI_NONE is always mapped to SET_HEAD with P1 = -1
* DO_CONDITION_YAW with P1 = 0 and P4 = 0 is mapped to SET_HEAD P1 = -1, any other combinations are mapped to SET_HEAD P1.
