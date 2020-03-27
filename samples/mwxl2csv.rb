#!/usr/bin/ruby
require 'xmlsimple'

m=XmlSimple.xml_in((ARGV[0] || STDIN), {'ForceArray' => false, 'KeepRoot' => false})
puts "no,wp,lat,lon,alt,p1,p2"
m['missionitem'].each do |i|
  p1 = i['action'] == 'WAYPOINT' ?  i['parameter1'].to_f/100 : i['parameter1'].to_i
  p2 = i['action'] == 'POSHOLD_TIME' ?  i['parameter2'].to_f/100 : i['parameter2'].to_i
  puts [i['no'], i['action'], i['lat'], i['lon'], i['alt'], p1,p2].join(',')
end
