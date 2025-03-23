#!/bin/bash


SCRIPT_DIR=$(dirname $0)
EXEC_DIR=$pwd

handle_interrupt() {
    cd $EXEC_DIR
    exit 0
}

cd $(realpath "${SCRIPT_DIR}/backend/src")

trap handle_interrupt SIGINT

go run main.go

cd $EXEC_DIR
