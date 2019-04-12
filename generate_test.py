#!/bin/python3
import datetime, math, sys
start = datetime.datetime.now()- datetime.timedelta(minutes=10)
start = start.replace(microsecond=0, second=0, minute=0, hour=0)

end = start + datetime.timedelta(days=28)
interval = datetime.timedelta(minutes=10)
daylength = datetime.timedelta(hours=24)

sys.stdout.write("datetime,\t\tdatetime-sim,\t\ttemp,\thum,\t0,\t1,\t3,\t4,\t5,\t6,\t7\n")
while start < end:
    start += interval
    phase = 0.5+ math.sin(start.timestamp()/daylength.total_seconds()*math.pi*2) /2
    # print(start, "\t{0:.1f}".format(lphase))
    # this is for psi lights, absolute
    lights = [int(max((phase-0.1)*1022*(.5+(math.sin(x/5)/3)),0)) for x in range(7)]
    # this is for heliospectra lights, percent
    # lights = [max((phase-0.1)*100*(.5+(math.sin(x/5)/3)),0) for x in range(7)]
    #lights[-1] = 10
    # this is for inbuild chamber lights, 2x values usually.
    # lights = [min(int(phase * 6),5)]*2
    temp = 23+math.sin(phase)*10
    hum = 85-math.sin(phase)*75
    x = [start.isoformat(),(start+datetime.timedelta(hours=4)).isoformat(), int(temp), hum]+lights
    # chamber
    # sys.stdout.write("{0},\t{1},\t{2:.2f},\t{3:.2f},\t{4},\t{5}\n".format(*x))
    # heliospectra
    # sys.stdout.write("{0},\t{1},\t{2:.2f},\t{3:.2f},\t{4:.2f},\t{5:.2f},\t{6:.2f},\t{7:.2f},\t{8:.2f},\t{9:.2f},\t{10:.2f}\n".format(*x))
    # psi
    sys.stdout.write("{0},\t{1},\t{2:.2f},\t{3:.2f},\t{4},\t{5},\t{6},\t{7},\t{8},\t{9},\t{10}\n".format(*x))
