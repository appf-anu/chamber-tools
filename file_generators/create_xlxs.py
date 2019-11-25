#!/bin/python3
import datetime
import math
from openpyxl import Workbook
wb = Workbook(write_only=True)
ws = wb.create_sheet(title="timepoints")

start = datetime.datetime.now() - datetime.timedelta(minutes=10)
start = start.replace(microsecond=0, second=0, minute=0, hour=0)

channels = [1, 2, 3, 4, 5, 6, 7, 8]
maxValue = 100

end = start + datetime.timedelta(days=3)
interval = datetime.timedelta(minutes=1)
# daylength = datetime.timedelta(hours=3)

ws.append(["datetime", "datetime_sim", "humidity", "temperature", "light1", "light2"])

x = 0


def is_night(dt):
    return not (7 <= dt.hour < 23)


a = 0
while start < end:
    a += 1
    dt = start

    if is_night(dt):
        start += datetime.timedelta(seconds=10)
        light1 = light2 = 0
        temp = 20
    else:
        temp = 28
        if a % 2 == 0:
            start += datetime.timedelta(seconds=10)
            light1 = light2 = 2

        else:
            light1 = 5
            light2 = 3
            start += datetime.timedelta(seconds=2)

    ws.append([
        dt,
        dt,
        55,
        temp,
        light1,
        light2
    ])

wb.save(filename="timepoints.xlsx")
