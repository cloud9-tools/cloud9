#!/usr/bin/make -f

all: model.pb.go

clean:
	rm -f *.pb.go

%.pb.go: %.proto
	protoc --go_out=. $<

.PHONY: all clean
