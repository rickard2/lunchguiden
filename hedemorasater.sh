#!/bin/bash

WEEK=`date +%W`

/home/rickard/Dropbox/Kod/go/lunchguiden/lunchguiden -url="http://service.dt.se/lunch/lunch.asp?tidning=sdt&vecka=$WEEK" -out="/var/www/lunchguiden/hedemorasater.v$WEEK.json" -week=$WEEK -city="Hedemora/Sater"
