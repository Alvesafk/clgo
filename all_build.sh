#!/bin/bash 

# Copyright (c) 2026 Alvesafk. All Rights Reserved.
#
# all_build.sh was made to help me with compiling the project into a normal Linux ELF binary
# and also an .exe file for Windows. It compiles first for Windows using the env variables
# as GOOS=windows and GOARCH=amd64, then to Linux with GOOS=linux and GOARCH=amd64. The bins
# are stored in a bin directory created by the shell.
#
# Make sure, if you are going to run it, run it on the same directory that the script is.

echo "Starting compilation"
if [ ! -d "bin/" ]; then
	echo "No bin folder was found."
	mkdir -v bin/
else
	echo "Bin folder was found."
	echo "Content will be removed in 5 seconds, interrupt the program if you are not sure."
	
	sleep 5

	echo "Removing content."
	rm bin/*
	echo "Removed content."
fi 

echo "Compiling for Windows / .exe"
GOOS=windows GOARCH=amd64 go build -o bin/clgo.exe .
echo "Finished"

echo "Compiling for Linux / elf"
GOOS=linux GOARCH=amd64 go build -o bin/clgo .

echo "All done."
