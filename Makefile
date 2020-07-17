# This Makefile is just for development purposes

.PHONY: default clean glow run log

default: glow

clean:
	rm -f ./glow

glow:
	go build

run: clean glow
	./glow

log:
	GLOW_UI_LOGFILE=debug.log
	> debug.log
	tail -f debug.log
