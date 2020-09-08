# This Makefile is just for development purposes

.PHONY: default clean glow run log

LOGFILE := debug.log

default: glow

clean:
	rm -f ./glow

glow:
	go build

run: clean glow
	GLOW_LOGFILE=$(LOGFILE) ./glow

log:
	> $(LOGFILE)
	tail -f $(LOGFILE)
