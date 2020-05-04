#!/bin/sh

export LOG_LEVEL=trace
export NAME=cicd-lmz
export PORT=9000
export VERSION=1.0.3
export SLEEP=30
export CRON="0/2 * * * *"

${1}
