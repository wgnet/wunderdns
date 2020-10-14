all:
	mkdir -p bin
	go test github.com/wgnet/wunderdns/wunderdns
	go test github.com/wgnet/wunderdns/httpapi
	go build -o bin/wunderdns


.PHONY: all
