#!/bin/bash

WEEK=`date +%W`

/home/rickard/Dropbox/Kod/go/lunchguiden/lunchguiden -url="http://service.dt.se/lunch/lunch.asp?tidning=mt&vecka=$WEEK" -out="/var/www/lunchguiden/mora.v$WEEK.json" -week=$WEEK -city="Mora"
