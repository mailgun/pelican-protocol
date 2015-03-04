
.phony: all debug run

all:
	go install
	cd client-tcp-over-http; go build -o client-tcp-over-http; cp -p client-tcp-over-http ../bin
	cd server-tcp-over-http; go build -o server-tcp-over-http; cp -p server-tcp-over-http ../bin
	cd send; go build -o send; cp -p send ../bin
	cd dest; go build -o dest; cp -p dest ../bin

debug:
	go install  -gcflags "-N -l" 
	cd client-tcp-over-http; go build -gcflags "-N -l"  -o client-tcp-over-http; cp -p client-tcp-over-http
	cd server-tcp-over-http; go build -gcflags "-N -l"  -o server-tcp-over-http; cp -p server-tcp-over-http

run:
	pkill -9 client-tcp-over-http || echo
	pkill -9 server-tcp-over-http || echo
	bin/server-tcp-over-http &
	bin/client-tcp-over-http &

clean:
	rm -f *~
	pkill -9 client-tcp-over-http || echo
	pkill -9 server-tcp-over-http || echo

testbuild:
	go test -c -gcflags "-N -l" -v

