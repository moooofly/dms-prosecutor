#!/usr/bin/env bash

PROSECUTOR_WORKING_DIR=$(pwd)

PROSECUTOR_BIN_NAME='dms-prosecutor'

PROSECUTOR_PID_FILE_DIR='/run'
PROSECUTOR_PID_FILE_NAME='dms-prosecutor.pid'
PROSECUTOR_PID_FILE=${PROSECUTOR_PID_FILE_DIR}/${PROSECUTOR_PID_FILE_NAME}

PROSECUTOR_EXE=${PROSECUTOR_WORKING_DIR}'/'${PROSECUTOR_BIN_NAME}

PROSECUTOR_CFG=$2


case $1 in
start)
    echo "starting prosecutor..."
    echo "using configure file: ${PROSECUTOR_CFG}"

    if [ -f ${PROSECUTOR_PID_FILE} ]; then
        if kill -0 $(cat ${PROSECUTOR_PID_FILE}) > /dev/null 2>&1; then
            pid=$(cat ${PROSECUTOR_PID_FILE})
            echo "prosecutor already running as process ${pid}"
            exit -1
        else
            echo "pid file exists but process does not exist, cleaning pid file"
            rm -f ${PROSECUTOR_PID_FILE}
        fi
    fi

    ${PROSECUTOR_EXE} --conf=${PROSECUTOR_CFG} &

    if [ $? -eq 0 ]; then
        pid=$!
        if echo ${pid} > ${PROSECUTOR_PID_FILE}; then
            echo "prosecutor started, as pid ${pid}"
            exit 0
        else
            echo "failed to write pid file"
            exit -1
        fi
    else
        echo "prosecutor failed to start"
        exit -1
    fi
    ;;

stop)
    echo "stopping prosecutor..."

    if [ ! -f ${PROSECUTOR_PID_FILE} ]; then
        echo "no prosecutor to stop (could not find file ${PROSECUTOR_PID_FILE})"
        exit -1
    else
        kill -15 $(cat ${PROSECUTOR_PID_FILE})
        rm -f ${PROSECUTOR_PID_FILE}
        echo "prosecutor stopped"
        exit 0
    fi
    ;;

status)
    if [ ! -f ${PROSECUTOR_PID_FILE} ]; then
        echo "no prosecutor running"
        exit -1
    else
        if kill -0 $(cat ${PROSECUTOR_PID_FILE}) > /dev/null 2>&1; then
            pid=$(cat ${PROSECUTOR_PID_FILE})
            echo "prosecutor running, as pid ${pid}"
            exit 0
        else
            echo "pid file exists but process does not exist, cleaning pid file"
            rm -f ${PROSECUTOR_PID_FILE}
            exit -1
        fi
    fi
    ;;

version)
    ${PROSECUTOR_EXE} --version
    ;;

*)
    echo "usage: $0 {start|stop|status|version}" >&2

esac
