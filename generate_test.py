#!/bin/python3
import datetime
import math
import sys

start = datetime.datetime.now() - datetime.timedelta(minutes=10)
start = start.replace(microsecond=0, second=0, minute=0, hour=0)

channels = [1, 2, 3, 4, 5, 6, 7, 8]
maxValue = 100

end = start + datetime.timedelta(days=365)
interval = datetime.timedelta(minutes=1)
# daylength = datetime.timedelta(hours=3)

sys.stdout.write("datetime,\tdatetime-sim,\thumidity,\ttemperature,\tlight1,\tlight2\n")
# for x in channels:
#     sys.stdout.write(",\tchannel-{}".format(x))
fmt_line = "{dt},\t{dt_sim},\t{humidity},\t{temperature},\t{light1},\t{light2}\n"
x = 0


def is_night(dt):
    return not (7 <= dt.hour < 23)


a = 0
while start < end:
    a += 1
    dt = start

    if is_night(dt):
        start += datetime.timedelta(minutes=10)
        light1 = light2 = 0
        temp = 20
    else:
        temp = 28
        if a % 2 == 0:
            start += datetime.timedelta(minutes=10)
            light1 = light2 = 2

        else:
            light1 = 5
            light2 = 3
            start += datetime.timedelta(minutes=2)
    # print(dt, temp, light1, light2, is_night(dt))
    sys.stdout.write(fmt_line.format(
        dt=dt.isoformat(),
        dt_sim=dt.isoformat(),
        humidity=55,
        temperature=temp,
        light1=light1,
        light2=light2
    ))
    sys.stdout.flush()
    # phase = 0.5 + math.sin(start.timestamp() / daylength.total_seconds() * math.pi * 2) / 2
    # # print(start, "\t{0:.1f}".format(lphase))
    # # this is for psi lights, absolute
    # lights = [max((phase - 0.1) * maxValue * (.5 + (math.sin(x / 5) / 3)), 0) for x in channels]
    # # this is for heliospectra lights, percent
    # # lights = [max((phase-0.1)*maxValue*(.5+(math.sin(x/5)/3)),0) for x in channels]
    # # lights[-1] = 10
    # # this is for inbuild chamber lights, 2x values usually.

    # #lights = [min(int(phase * 6),5)]*(nChannels+1)

    # temp = 23 + math.sin(phase) * 10
    # hum = 85 - math.sin(phase) * 75
    # x = [start.isoformat(), (start + datetime.timedelta(hours=4)).isoformat(), int(temp), hum]
    # sys.stdout.write("{0},\t{1},\t{2:.2f},\t{3:.2f},\t".format(*x))
    # sys.stdout.write(",\t".join("{:.2f}".format(n) for n in lights))
    # sys.stdout.write("\n")
