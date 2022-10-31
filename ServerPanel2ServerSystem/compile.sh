#!/bin/sh

GOOS=darwin GOARCH=arm64 go build -o binaries/ServerPanel2ServerSystem.Mac-arm64-m1
GOOS=darwin GOARCH=amd64 go build -o binaries/ServerPanel2ServerSystem.Mac-x86-intel
GOOS=windows GOARCH=amd64 go build -o binaries/ServerPanel2ServerSystem.Win-amd64.exe
GOOS=windows GOARCH=386 go build -o binaries/ServerPanel2ServerSystem.Win-386.exe
GOOS=linux GOARCH=amd64 go build -o binaries/ServerPanel2ServerSystem.Linux-amd64
GOOS=linux GOARCH=386 go build -o binaries/ServerPanel2ServerSystem.Linux-386

cd binaries

zip ServerPanel2ServerSystem.Mac.zip ServerPanel2ServerSystem.Mac-arm64-m1 ServerPanel2ServerSystem.Mac-x86-intel 
zip ServerPanel2ServerSystem.Win.zip ServerPanel2ServerSystem.Win-amd64.exe ServerPanel2ServerSystem.Win-386.exe 
zip ServerPanel2ServerSystem.Linux.zip ServerPanel2ServerSystem.Linux-amd64 ServerPanel2ServerSystem.Linux-386

rm ServerPanel2ServerSystem.Win-amd64.exe ServerPanel2ServerSystem.Win-386.exe ServerPanel2ServerSystem.Linux-amd64 ServerPanel2ServerSystem.Linux-386 ServerPanel2ServerSystem.Mac-arm64-m1 ServerPanel2ServerSystem.Mac-x86-intel

cd ..
