# This Makefile is just for development purposes

.PHONY: default clean glow run log

default: glow

clean:
	rm -f ./glow

glow:
	go build
	cp glow.desktop /usr/share/applications/

run: clean glow
	./glow

log:
	tail -f ~/.cache/glow/glow.log
