#!/bin/python3
import datetime
import random
from openpyxl import Workbook

lights_headers = {
    "psi": [
        "channel-1",  # white
        "channel-2",  # blue
        "channel-3",  # green
        "channel-4",  # nearest red
        "channel-5",  # near red
        "channel-6",  # far red
        "channel-7",  # infra red
        "channel-8"   # unknown, further infra red?
    ],
    "heliospectra_s7": [
        "channel-1",  # 400nm
        "channel-2",  # 420nm
        "channel-3",  # 450nm
        "channel-4",  # 530nm
        "channel-5",  # 630nm
        "channel-6",  # 660nm
        "channel-7"   # 735nm
    ],
    "heliospectra_s10": [
        "channel-1",  # 370nm
        "channel-2",  # 400nm
        "channel-3",  # 420nm
        "channel-4",  # 450nm
        "channel-5",  # 530nm
        "channel-6",  # 620nm
        "channel-7",  # 660nm
        "channel-8",  # 735nm
        "channel-9",  # 850nm
        "channel-10"  # 6500k
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

    channels_scaling = settings.get("channels_scaling", [])
    if len(channels_scaling) > 0:
        if len(channels_scaling) == len(lights_headers[lights]):
            fname += f"-{channels_scaling}".replace(" ", "")

    while start < end:
        temp = day_temp
        if is_night(start, day_start=day_start, day_end=day_end):
            temp = night_temp
        channels = [100] * len(lights_headers[lights])

        if len(channels_scaling) > 0:
            if len(channels_scaling) == len(channels):
                channels = [ch * sf for ch, sf in zip(channels, channels_scaling)]
            else:
                print(f"len(channels_scaling) == {len(channels_scaling)} != len(channels) =={len(channels)}")

        if is_night(start, day_start=day_start, day_end=day_end):
            channels = [0] * len(lights_headers[lights])

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
    fn = f"{fname}.xlsx".replace(" ", "")
    print(f'saving {fn}')
    wb.save(filename=fn)


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


settings = {
    "interval_m": 0.1,
    "day_temp": 28,
    "night_temp": 20,
    "day_start": 7,
    "day_end": 17,
    "humidity": 55
}

settings['lights'] = "conviron"

generate(settings)


# settings for ch36 2020-02-24
settings = {
    "lights": "psi",
    "interval_m": 10,
    "day_temp": 28,
    "night_temp": 22,
    "day_start": 6,
    "day_end": 22,
    "humidity": 60,
    "channels_scaling": [
        1.0,  # white
        1.0,  # blue
        1.0,  # green
        0.6,  # nearest red
        0.6,  # near red
        0.6,  # far red
        0.0,  # infra red
        0.0,  # unknown, further infra red?
    ]
}

generate(settings)

settings = {
    "interval_m": 10,
    "day_temp": 10,
    "night_temp": 7,
    "day_start": 9,
    "day_end": 23,
    "humidity": 55
}

settings['lights'] = "heliospectra_s7"

generate(settings)
