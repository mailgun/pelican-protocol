
.PHONY: all test clean

debug:
	go build -gcflags "-N -l"
	go install -gcflags "-N -l"

clean:
	rm -f *~ 

dock-bootstrap:
	docker build -t pel3 .
	docker run pel3 /sbin/my_init  --enable-insecure-key

dockcommit:
	docker commit 1a0785328b18  mailgun/pelican01

dock:
	docker run mailgun/pelican02 /sbin/my_init

dock-to-tar:
	docker save mailgun/pelican02 | gzip > pelican01-docker.tar.gz

# edit the name in the repositories file to change the image name
# I changed it to pelican02 to make sure it was reflected in 'docker images'. It was.
dock-from-tar:
	zcat pelican01-docker.tar.gz | docker load

docker-push:
	docker login
	docker commit cfca0bc349f3 jaten/pelican03
	#cff7026d259762268f623c77b403951699115b10a15807d26d14463f74c1f114
	docker push jaten/pelican03

## pushing jaten/pelica03 resulted in this url:
## Pushing tag for rev [cff7026d2597] on {https://cdn-registry-1.docker.io/v1/repositories/jaten/pelican03/tags/latest}
