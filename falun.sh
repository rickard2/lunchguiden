#!/bin/bash

WEEK=`date +%W`

/home/rickard/Dropbox/Kod/go/lunchguiden/lunchguiden -url="http://service.dt.se/lunch/lunch.asp?tidning=fk&vecka=$WEEK" -week=$WEEK -out="/var/www/lunchguiden/falun.v$WEEK.json" -city="Falun"
