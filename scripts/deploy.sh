#!/bin/bash

CONTAINER=user-stats
REPO=severeone/user-stats

docker stop ${CONTAINER} > /dev/null
docker rename ${CONTAINER} ${CONTAINER}-old > /dev/null

docker tag ${REPO}:latest ${REPO}:old > /dev/null

docker run -d --restart always --name ${CONTAINER} --net=host -v ~/user-stats/config.yml:/app/config.yml ${REPO}:latest

if [ $? -eq 0 ]; then
    docker rm ${CONTAINER}-old
    docker rmi ${REPO}:old > /dev/null
    echo "SUCCESS"
else
    docker tag ${REPO}:old ${REPO}:latest > /dev/null
    docker rmi ${REPO}:old > /dev/null
    docker rename ${CONTAINER}-old ${CONTAINER}
    docker start ${CONTAINER}
    echo "FAILURE"
    exit 1
fi
