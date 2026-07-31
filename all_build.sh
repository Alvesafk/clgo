#!/bin/bash 

# Copyright (c) 2026 Alvesafk. All Rights Reserved.
#
# all_build.sh was made to help me with compiling the project into a normal Linux ELF binary
# and also an .exe file for Windows. It compiles first for Windows using the env variables
# as GOOS=windows and GOARCH=amd64, then to Linux with GOOS=linux and GOARCH=amd64. The bins
# are stored in a bin directory created by the shell.
#
# Make sure, if you are going to run it, run it on the same directory that the script is.
set -uo pipefail
trap 'echo "Interrupted by user"; exit 130' INT

check() {
	if [ $? -ne 0 ]; then
		echo "Error: $1" >&2
		exit 1
	fi
}

echo "Starting compilation"

if [ ! -d "bin/" ]; then
	echo "No bin folder was found."
	mkdir -v bin/
	check "Failed to create bin/ folder"
else
	echo "Bin folder was found."
	echo "Content will be removed in 5 seconds, interrupt the program if you are not sure."
	sleep 5

	echo "Removing content."
	rm -f bin/*
	check "Failed to remove bin/ folder content"
	echo "Removed content."
fi

go mod verify
check "go mod verify failed"

go mod tidy
check "go mod tidy failed"

echo "Compiling for Windows / .exe"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -a -o bin/clgo_amd64.exe .
check "Windows build failed"
echo "Finished (Windows)"

echo "Compiling for Linux / elf"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -a -o bin/clgo_amd64 .
check "Build para Linux falhou"
echo "Finished (Linux)"

echo "All done."
