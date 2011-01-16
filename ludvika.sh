#!/bin/bash

WEEK=`date +%W`

/home/rickard/Dropbox/Kod/go/lunchguiden/lunchguiden -url="http://service.dt.se/lunch/lunch.asp?tidning=nlt&vecka=$WEEK" -out="/var/www/lunchguiden/ludvika.v$WEEK.json" -week=$WEEK -city="Ludvika"
