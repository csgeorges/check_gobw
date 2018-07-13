#!/bin/bash
go build -o check_gobw
GOARCH=386 go build -o check_gobw32
