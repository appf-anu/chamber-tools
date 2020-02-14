#!/bin/python3
import datetime
import random
from openpyxl import Workbook

lights_headers = {
    "psi": [
        "channel-1",
        "channel-2",
        "channel-3",
        "channel-4",
        "channel-5",
        "channel-6",
        "channel-7",
        "channel-8"
    ],
    "heliospectra_s7": [
        "channel-1",
        "channel-2",
        "channel-3",
        "channel-4",
        "channel-5",
        "channel-6",
        "channel-7"
    ],
    "heliospectra_s10": [
        "channel-1",
        "channel-2",
        "channel-3",
        "channel-4",
        "channel-5",
        "channel-6",
        "channel-7",
        "channel-8",
        "channel-9",
        "channel-10"
    ],
    "conviron": [
        "light1",
        "light2"
    ]

}


def is_night(dt, day_start=7, day_end=17):
    return not (day_start <= dt.hour < day_end)


def generate(settings: dict):

    wb = Workbook(write_only=True)
    ws = wb.create_sheet(title="timepoints")
    fname = ""

    # datetime has 19 characters
    ws.column_dimensions['A'].width = 19
    ws.column_dimensions['B'].width = 19
    lights = settings.get("lights", "conviron")
    ws.append(["datetime", "datetime-sim", "humidity", "temperature"] + lights_headers[lights])

    start = datetime.datetime.now()
    start = start.replace(microsecond=0, second=0, minute=0, hour=0)

    length_days = settings.get('length_days', 2)
    end = start + datetime.timedelta(days=length_days)
    interval_m = settings.get("interval_m", 10)
    interval = datetime.timedelta(minutes=interval_m)
    day_start = settings.get("day_start", 7)
    day_end = settings.get("day_end", 23)

    fname += f"conditions/{interval_m}m-for-{length_days}d-{day_start:02d}00to{day_end:02d}00_"

    day_temp = settings.get("day_temp", 28)
    night_temp = settings.get("night_temp", 20)
    fname += f"{day_temp}C-{night_temp}C_"

    humidity = settings.get("humidity", 55)
    fname += f"{humidity}rh_"

    fname += f"{lights}"

    while start < end:
        temp = day_temp
        if is_night(start, day_start=day_start, day_end=day_end):
            temp = night_temp

        channels = [100] * len(lights_headers[lights])
        if is_night(start, day_start=day_start, day_end=day_end):
            channels = [0] * len(lights_headers[lights])

        # channels = [random.uniform(0, 100) for _ in range(8)]
        ws.append([
            start,
            start,
            55,
            temp
        ] + channels)
        start += interval

    if settings.get("filename"):
        print(f'saving {settings.get("filename")}')
        wb.save(filename=settings.get("filename"))
        return

    wb.save(filename=f"{fname}.xlsx")
    print(f'saving {fname}.xlsx')


settings = {
    "interval_m": 10,
    "day_temp": 28,
    "night_temp": 20,
    "day_start": 7,
    "day_end": 23,
    "humidity": 55


}

settings['lights'] = "heliospectra_s10"

generate(settings)

settings['lights'] = "heliospectra_s7"

generate(settings)

settings['lights'] = "psi"

generate(settings)


settings = {
    "interval_m": 10,
    "day_temp": 28,
    "night_temp": 20,
    "day_start": 7,
    "day_end": 17,
    "humidity": 55
}

settings['lights'] = "heliospectra_s10"

generate(settings)

settings['lights'] = "heliospectra_s7"

generate(settings)

settings['lights'] = "psi"

generate(settings)


settings = {
    "interval_m": 60,
    "day_temp": 21,
    "night_temp": 21,
    "day_start": 9,
    "day_end": 21,
    "humidity": 55
}

settings['lights'] = "heliospectra_s10"

generate(settings)

settings['lights'] = "heliospectra_s7"

generate(settings)

settings['lights'] = "psi"

generate(settings)


settings = {
    "interval_m": 10,
    "day_temp": 21,
    "night_temp": 21,
    "day_start": 9,
    "day_end": 21,
    "humidity": 55
}

settings['lights'] = "heliospectra_s10"

generate(settings)

settings['lights'] = "heliospectra_s7"

generate(settings)

settings['lights'] = "psi"

generate(settings)
