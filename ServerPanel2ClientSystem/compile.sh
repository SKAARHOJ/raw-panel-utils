#!/bin/sh

GOOS=darwin GOARCH=arm64 go build -o binaries/ServerPanel2ClientSystem.Mac-arm64-m1
GOOS=darwin GOARCH=amd64 go build -o binaries/ServerPanel2ClientSystem.Mac-x86-intel
GOOS=windows GOARCH=amd64 go build -o binaries/ServerPanel2ClientSystem.Win-amd64.exe
GOOS=windows GOARCH=386 go build -o binaries/ServerPanel2ClientSystem.Win-386.exe
GOOS=linux GOARCH=amd64 go build -o binaries/ServerPanel2ClientSystem.Linux-amd64
GOOS=linux GOARCH=386 go build -o binaries/ServerPanel2ClientSystem.Linux-386

cd binaries

zip ServerPanel2ClientSystem.Mac.zip ServerPanel2ClientSystem.Mac-arm64-m1 ServerPanel2ClientSystem.Mac-x86-intel 
zip ServerPanel2ClientSystem.Win.zip ServerPanel2ClientSystem.Win-amd64.exe ServerPanel2ClientSystem.Win-386.exe 
zip ServerPanel2ClientSystem.Linux.zip ServerPanel2ClientSystem.Linux-amd64 ServerPanel2ClientSystem.Linux-386

rm ServerPanel2ClientSystem.Win-amd64.exe ServerPanel2ClientSystem.Win-386.exe ServerPanel2ClientSystem.Linux-amd64 ServerPanel2ClientSystem.Linux-386 ServerPanel2ClientSystem.Mac-arm64-m1 ServerPanel2ClientSystem.Mac-x86-intel

cd ..
