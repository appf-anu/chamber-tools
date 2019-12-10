#!/bin/python3
from openpyxl.styles import NamedStyle
import datetime
import math
from openpyxl import Workbook
import random
wb = Workbook(write_only=True)
ws = wb.create_sheet(title="timepoints")
# datetime has 19 characters
ws.column_dimensions['A'].width = 19
ws.column_dimensions['B'].width = 19


start = datetime.datetime.now()

start = start.replace(microsecond=0, second=0, minute=0, hour=0)

channels = [1, 2, 3, 4, 5, 6, 7, 8]
maxValue = 100

end = start + datetime.timedelta(days=7)
interval = datetime.timedelta(minutes=1)


# ws.append(["datetime", "datetime-sim", "humidity", "temperature", "light1", "light2"])

channels_psi = ["channel-1",
                "channel-2",
                "channel-3",
                "channel-4",
                "channel-5",
                "channel-6",
                "channel-7",
                "channel-8"]
ws.append(["datetime", "datetime-sim", "humidity", "temperature"] + channels_psi)

x = 0


def is_night(dt):
    return not (7 <= dt.hour < 23)


while start < end:
    if is_night(start):
        # start += datetime.timedelta(minutes=10)
        light1 = light2 = 0
        temp = 20 if start.minute % 2 == 0 else 25
    else:
        temp = 28 if start.minute % 2 == 0 else 24

    channels = [random.uniform(0, 100) for _ in range(8)]
    ws.append([
        start,
        start,
        55,
        temp
    ] + channels)
    start += interval


wb.save(filename="conditions.xlsx")
