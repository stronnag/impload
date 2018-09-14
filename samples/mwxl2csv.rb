#!/usr/bin/ruby
require 'xmlsimple'

m=XmlSimple.xml_in((ARGV[0] || STDIN), {'ForceArray' => false, 'KeepRoot' => false})
puts "no,wp,lat,lon,alt,p1"
m['MISSIONITEM'].each do |i|
  p1 = i['action'] == 'WAYPOINT' ?  i['parameter1'].to_f/100 : 0
  puts [i['no'], i['action'], i['lat'], i['lon'], i['alt'], p1].join(',')
end
