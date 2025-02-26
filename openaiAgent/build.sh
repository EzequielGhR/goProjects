#!/bin/bash


DIRNAME=$(dirname $0)
EXECNAME=$pwd

echo $DIRNAME
echo $EXECNAME

cd $DIRNAME
mkdir -p bin/v1

cd src/main

go build -o ../../bin/v1/main.o

cd $EXECNAME