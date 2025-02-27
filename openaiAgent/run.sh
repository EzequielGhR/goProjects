#!/bin/bash


DIRNAME=$(realpath $(dirname $0))
EXECNAME=$pwd

if [ -z "${1}" ]
    then echo "Missing prompt"
    exit 1
fi

cd $DIRNAME

if [ ! -f "bin/v1/main.o" ]
    then ./build.sh || { 
        cd $EXECNAME
        echo "Failed build"
        exit 1
    }
fi

cd bin/v1

./main.o "${1}"

st=$?

cd $EXECNAME

if [ $st -eq 0 ]
    then echo "Success"
    exit 0
fi

echo "Failed run"
