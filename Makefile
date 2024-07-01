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
	tail -f ~/.cache/glow/glow.log
