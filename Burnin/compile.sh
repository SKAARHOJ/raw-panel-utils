#!/bin/sh

GOOS=darwin GOARCH=arm64 go build -o binaries/Burnin.Mac-arm64-m1
GOOS=darwin GOARCH=amd64 go build -o binaries/Burnin.Mac-x86-intel
GOOS=windows GOARCH=amd64 go build -o binaries/Burnin.Win-amd64.exe
GOOS=windows GOARCH=386 go build -o binaries/Burnin.Win-386.exe
GOOS=linux GOARCH=amd64 go build -o binaries/Burnin.Linux-amd64
GOOS=linux GOARCH=386 go build -o binaries/Burnin.Linux-386

cd binaries

zip Burnin.Mac.zip Burnin.Mac-arm64-m1 Burnin.Mac-x86-intel 
zip Burnin.Win.zip Burnin.Win-amd64.exe Burnin.Win-386.exe 
zip Burnin.Linux.zip Burnin.Linux-amd64 Burnin.Linux-386

rm Burnin.Win-amd64.exe Burnin.Win-386.exe Burnin.Linux-amd64 Burnin.Linux-386 Burnin.Mac-arm64-m1 Burnin.Mac-x86-intel

cd ..
