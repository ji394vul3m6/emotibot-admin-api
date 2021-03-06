#!/bin/bash

# number of w-v-emotion(worker-voice-emotion-analysis)
num_of_worker_analysis=$NUM_ANA_WORKER
if [ "$num_of_worker_analysis" == "" ]; then
	num_of_worker_analysis=5
fi
echo "num_of_worker_analysis: $num_of_worker_analysis"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

source $1
if [ "$?" -ne 0 ]; then
  echo "Erorr, can't open envfile: $1"
  echo "Usage: $0 <env file>"
  echo "e.g., "
  echo " $0 api-sh.env [SERVICE1] [SERVICE2]..."
  exit 1
else
  envfile=$1
  echo "# Using envfile: $envfile"
fi
shift

if [ "$envfile" == "test.env" ]; then
    mkdir -p /tmp/persistant_storage
else
    mkdir -p /home/deployer/persistant_storage
fi

while [ $# != 0 ]
do
    echo $1
    if [ "$1" == "w-v-emotion" ]; then
        scale="--scale $1=$num_of_worker_analysis"
    fi
    service="$service "$1
    shift
done

if [ "$service" == "" ]; then
    scale="--scale w-v-emotion=$num_of_worker_analysis"
fi
# prepare docker-compose env file
cp $envfile .env

docker-compose -f ./docker-compose.yml rm -s $service
cmd="docker-compose -f ./docker-compose.yml up --force-recreate --remove-orphans -d $scale $service" 
echo $cmd
eval $cmd
