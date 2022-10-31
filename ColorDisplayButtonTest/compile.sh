#!/bin/sh

GOOS=darwin GOARCH=arm64 go build -o binaries/ColorDisplayButtonTest.Mac-arm64-m1
GOOS=darwin GOARCH=amd64 go build -o binaries/ColorDisplayButtonTest.Mac-x86-intel
GOOS=windows GOARCH=amd64 go build -o binaries/ColorDisplayButtonTest.Win-amd64.exe
GOOS=windows GOARCH=386 go build -o binaries/ColorDisplayButtonTest.Win-386.exe
GOOS=linux GOARCH=amd64 go build -o binaries/ColorDisplayButtonTest.Linux-amd64
GOOS=linux GOARCH=386 go build -o binaries/ColorDisplayButtonTest.Linux-386

cd binaries

zip ColorDisplayButtonTest.Mac.zip ColorDisplayButtonTest.Mac-arm64-m1 ColorDisplayButtonTest.Mac-x86-intel 
zip ColorDisplayButtonTest.Win.zip ColorDisplayButtonTest.Win-amd64.exe ColorDisplayButtonTest.Win-386.exe 
zip ColorDisplayButtonTest.Linux.zip ColorDisplayButtonTest.Linux-amd64 ColorDisplayButtonTest.Linux-386

rm ColorDisplayButtonTest.Win-amd64.exe ColorDisplayButtonTest.Win-386.exe ColorDisplayButtonTest.Linux-amd64 ColorDisplayButtonTest.Linux-386 ColorDisplayButtonTest.Mac-arm64-m1 ColorDisplayButtonTest.Mac-x86-intel

cd ..
