backend storage
==============

camlistore : our eventual target; especially for user's keys.
User's set of keys should easily fit in memory, so performance/space
should not be a problem.
There are iphone and android applications for camlistore already,
and camlistore will have an SMTP interface, so we should be
able to get sane replication/backup strategy in place for
users

Backups and perforance for the server side: we may need to be fast
to support websites with lots of traffic; start with straight 
(memory mapped) boltdb for now.


camlistore notes:
--------------------
~~~
git clone https://camlistore.googlesource.com/camlistore
cd into that dir
make
camlistored # generate certs

there are already fuse, iphone, android clients. yes!

~~~~