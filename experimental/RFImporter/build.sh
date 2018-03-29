#!/bin/bash
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

cmd="docker build -t docker-reg.emotibot.com.cn:55688/RFImporter -f $DIR/dockerfile $DIR"
echo $cmd
eval $cmd

