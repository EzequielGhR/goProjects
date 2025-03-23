#!/bin/bash


SCRIPT_DIR=$(dirname $0)
EXEC_DIR=$pwd

handle_interrupt() {
    cd $EXEC_DIR
    exit 0
}

cd $(realpath "${SCRIPT_DIR}/frontend/journaling-lit-app")

trap handle_interrupt SIGINT

npm run dev

cd $EXEC_DIR