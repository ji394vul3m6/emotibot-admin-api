#!/bin/sh

overwrite_file="
  robot.properties
  FRConfig.properties
"
DIR=$( cd -- "$(dirname -- "$0")" && pwd )
cd $DIR
for file in `find InitFiles -type f`
do
  target=`echo $file | sed "s#InitFiles#${ADMIN_SERVER_MOUNT_PATH}#g"`;
  target_dir=`dirname $target`
  if ! [[ -e $target ]];
  then
    mkdir -p $target_dir;
    cp $file $target;
    echo "[Files] Copy `basename $file` into $target"
  else
    for overwrite in $overwrite_file
    do
      if [[ `basename $file` == $overwrite ]];
      then
        time=`date +%s`;
        cp $target $target.bak.$time;
        cp $file $target;
        echo "[Files] Replace $target with $file";
        cmd="diff $target.bak.$time $target";
        echo $cmd;
        eval $cmd;
      fi
    done
  fi
done
