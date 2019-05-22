#!/usr/bin/python3
# -*- coding: utf-8 -*-

import time
from datetime import datetime

if __name__ == "__main__":
    while (True):
        curtime = datetime.now()
        try:
            print("Time: " + curtime.strftime('%Y/%m/%d %H:%M:%S'), flush=True)
        except (BrokenPipeError):
            pass
        time.sleep(1)
