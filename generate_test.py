#!/bin/python3
import datetime, math, sys
start = datetime.datetime.now()- datetime.timedelta(minutes=10)
start = start.replace(microsecond=0, second=0, minute=0, hour=0)

channels = [1,2,3,4,5,6,7,8]
maxValue = 100

end = start + datetime.timedelta(days=5)
interval = datetime.timedelta(minutes=10)
daylength = datetime.timedelta(hours=3)

sys.stdout.write("datetime,\tdatetime-sim,\thumidity,\ttemperature")
for x in channels:
    sys.stdout.write(",\tchannel-{}".format(x))
sys.stdout.write("\n")
while start < end:
    start += interval
    phase = 0.5+ math.sin(start.timestamp()/daylength.total_seconds()*math.pi*2) /2
    # print(start, "\t{0:.1f}".format(lphase))
    # this is for psi lights, absolute
    lights = [max((phase-0.1)*maxValue*(.5+(math.sin(x/5)/3)),0) for x in channels]
    # this is for heliospectra lights, percent
    # lights = [max((phase-0.1)*maxValue*(.5+(math.sin(x/5)/3)),0) for x in channels]
    # lights[-1] = 10
    # this is for inbuild chamber lights, 2x values usually.

    #lights = [min(int(phase * 6),5)]*(nChannels+1)

    temp = 23+math.sin(phase)*10
    hum = 85-math.sin(phase)*75
    x = [start.isoformat(),(start+datetime.timedelta(hours=4)).isoformat(), int(temp), hum]
    sys.stdout.write("{0},\t{1},\t{2:.2f},\t{3:.2f},\t".format(*x))
    sys.stdout.write(",\t".join("{:.2f}".format(n) for n in lights))
    sys.stdout.write("\n")
