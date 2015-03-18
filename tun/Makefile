
.phony: all debug run

all:
	go install
	cd client-tcp-over-http; go build -o client-tcp-over-http; cp -p client-tcp-over-http ../bin
	cd server-tcp-over-http; go build -o server-tcp-over-http; cp -p server-tcp-over-http ../bin
	cd send; go build -o send; cp -p send ../bin
	cd dest; go build -o dest; cp -p dest ../bin
	cd cmd/web; go build; cp -p web ../../bin
	cd cmd/rev; go build; cp -p rev ../../bin
	cd cmd/fwd; go build; cp -p fwd ../../bin

debug:
	go install  -gcflags "-N -l" 
	cd client-tcp-over-http; go build -gcflags "-N -l"  -o client-tcp-over-http; cp -p client-tcp-over-http ../bin
	cd server-tcp-over-http; go build -gcflags "-N -l"  -o server-tcp-over-http; cp -p server-tcp-over-http ../bin
	cd cmd/web; go build -gcflags "-N -l" ; cp -p web ../../bin
	cd cmd/rev; go build -gcflags "-N -l" ; cp -p rev ../../bin
	cd cmd/fwd; go build -gcflags "-N -l" ; cp -p fwd ../../bin

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

