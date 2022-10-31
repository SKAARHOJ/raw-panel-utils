#!/bin/sh

GOOS=darwin GOARCH=arm64 go build -o binaries/IPServer.Mac-arm64-m1 ipserver.go 
GOOS=darwin GOARCH=amd64 go build -o binaries/IPServer.Mac-x86-intel ipserver.go
GOOS=windows GOARCH=amd64 go build -o binaries/IPServer.Win-amd64.exe ipserver.go
GOOS=windows GOARCH=386 go build -o binaries/IPServer.Win-386.exe ipserver.go
GOOS=linux GOARCH=amd64 go build -o binaries/IPServer.Linux-amd64 ipserver.go
GOOS=linux GOARCH=386 go build -o binaries/IPServer.Linux-386 ipserver.go

GOOS=darwin GOARCH=arm64 go build -o binaries/IPClient.Mac-arm64-m1 ipclient.go
GOOS=darwin GOARCH=amd64 go build -o binaries/IPClient.Mac-x86-intel ipclient.go
GOOS=windows GOARCH=amd64 go build -o binaries/IPClient.Win-amd64.exe ipclient.go
GOOS=windows GOARCH=386 go build -o binaries/IPClient.Win-386.exe ipclient.go
GOOS=linux GOARCH=amd64 go build -o binaries/IPClient.Linux-amd64 ipclient.go
GOOS=linux GOARCH=386 go build -o binaries/IPClient.Linux-386 ipclient.go


cd binaries

zip IPServer.Mac.zip IPServer.Mac-arm64-m1 IPServer.Mac-x86-intel 
zip IPServer.Win.zip IPServer.Win-amd64.exe IPServer.Win-386.exe 
zip IPServer.Linux.zip IPServer.Linux-amd64 IPServer.Linux-386

rm IPServer.Win-amd64.exe IPServer.Win-386.exe IPServer.Linux-amd64 IPServer.Linux-386 IPServer.Mac-arm64-m1 IPServer.Mac-x86-intel

zip IPClient.Mac.zip IPClient.Mac-arm64-m1 IPClient.Mac-x86-intel 
zip IPClient.Win.zip IPClient.Win-amd64.exe IPClient.Win-386.exe 
zip IPClient.Linux.zip IPClient.Linux-amd64 IPClient.Linux-386

rm IPClient.Win-amd64.exe IPClient.Win-386.exe IPClient.Linux-amd64 IPClient.Linux-386 IPClient.Mac-arm64-m1 IPClient.Mac-x86-intel

cd ..
