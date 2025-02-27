#!/bin/bash


DIRNAME=$(realpath $(dirname $0))
EXECNAME=$pwd

mkdir -p $DIRNAME/bin/v1

cd $DIRNAME/src/agent
go mod tidy

cd $DIRNAME/src/tools
go mod tidy

cd $DIRNAME/src/trace
go mod tidy

cd $DIRNAME/src/main
go mod tidy

go build -o ../../bin/v1/main.o

cd $EXECNAME