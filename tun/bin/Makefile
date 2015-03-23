all:
	pkill -9 rev fwd || true
	cd ..; make all
	./rev -dest :22 -listen :8080 &
	./fwd --dest :8080 --listen :2222 &
