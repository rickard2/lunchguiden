#!/bin/bash

WEEK=`date +%W`

/home/rickard/Dropbox/Kod/go/lunchguiden/lunchguiden -url="http://service.dt.se/lunch/lunch.asp?tidning=bt&vecka=$WEEK" -out="/var/www/lunchguiden/borlange.v$WEEK.json" -week=$WEEK -city="Borlange"
