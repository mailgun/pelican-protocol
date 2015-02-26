package pelican

/*
Backups are the single most important part of being able to deploy pki solutions.

Because: as a server admin, if you loose your server key, nobody will be able to access their account again. There's no way to tell the difference between a MITM attack and the server suddenly having new keys because the old ones were lost.

As a browser user on the client end, if you loose your key then the server has no way of telling that you are you versus an attacker. You are locked out.


Do we want users to be able to backup their password protected keys with the server? quite possibly yes. Then the admin can send the client-user their password protected keys and give them access to their account again if need be.

How do servers backup their keys?

Instead of Certificate issuing business, we need a key-backup business.

In order to retreive a backup of your key, you would simply:
 - send an email, with a request that you be emailed back your archive (they are pretty small).
 -

To keep the amount of key material small, there should be a separate protocol for logins:
login-https:// which will indicate that }
login-http://  which will indicate that } in both cases, SSH should be used.


Backup-recovery assurance: require that two friends sign up with you, (e.g. so it can have viral coef > 1), and that either of them (or others you designate) can initiate re-keying of your keys and sending them to your emails on file.)  Has the possibility of being hacked, so how to avoid?

How can you submit a "I lost my password" request, a request that cannot be faked?

Nice to be able to do that without othe people's having to know about it.

IP addresses / Geolocations that are allowed to reset the account.

Get an email on with a list of password reset codes.

one-time burn codes: unwrap a password once, can't reset it again?

backup to the sky-net / bitcoin infrastructure / name-coin.

images: much easier for humans to validate than long streams of numbers; QR codes--easy for phone to read.

It may not be ultra secure but: it would still prevent most phishing attacks, and if we can break the association between the identity and the key, then
a) later we can upgrade the encryption used; and
b) we'll get adoption because recovery and rotation isn't so lacking that its a showstopper.

I like building in the viral/distribution into the key recovery procedure: send this archive to 5 people that you trust to only return it to you.  Security by obscurity is not really security.



simple 1-1 client-server much more appealing. So the server can send you your keys in an encrypted archive, and could also regularly send you other stuff... without having to know your.

Before the login/ "reset password" is ever encountered, download the keys from skynet/blockchain/requesting that mailgun send them back to you in an email. Can you make this request over an already secured channel?

"""TrueCrypt. does 3 layer:
AES-Twofish-Serpent and you can do them in just about any combination. And,
 you can use RIPEMD or SHA as a hashing algorithm."""


-----------------
backups... tied to a particular host/ip, yes? what if that ip changes? tied to a particular device?
then it becomes identifiable with that computer, no?/ identifies the computer.

If you don't give the MITM notice that the connection is going to be ssh-http, then he can't modify it to downgrade it. And there are a ton of 'http://' strings that don't want to be changed in web pages for back-compat reasons.

** feature todo: once-upgraded from HTTP->SSH, we should give a warning on downgrade back to HTTP at the same site
.

problem: traffic monitoring at gateways that won't let ssh traffic tunnel over http.
for proxy to work, we actually need to use TLS to connect for everything, otherwise firewalls will filter port 80 traffic that is not http. Or: can we tunnel
https://github.com/nf/gohttptun/ <- a tunnel over http by Andrew Gerrand.

http://stackoverflow.com/questions/21417223/simple-ssh-port-forward-in-golang


look at what ssh -D does: b/c it sets up a socks proxy.


goproxy for all requests (both http and https):
  -> https can get through okay, and we could tunnel ssh over https -> then back to the https (localhost :443) port.
  -> http seems to require additional work to tunnel ssh, like using Andrew Gerrand's ng/gohttuptun.


browser side:
https://github.com/nwjs/nw.js  (chromium + nodejs)
and
https://github.com/lonnc/golang-nw

examples (write your own browser solution)
https://github.com/nwjs/nw.js/wiki/List-of-apps-and-companies-using-nw.js


http://shadowsocks.org/en/index.html
A secure socks5 proxy

https://github.com/nihgwu/Nevermore <- wraps shadowsocks with a nw.js UI

https://github.com/CzarekTomczak/cef2go

-------------
start by describing the ideal.
--
then describe the barrier to adoption, and give the workaround.

https://github.com/inconshreveable/go-tunnel  ### doesn't actually do what we want.

http://http-tunnel.sourceforge.net/

http://sshh.sourceforge.net/

http://www.nocrew.org/software/httptunnel.html  <<-- will this work?!


////////////////////
// bosh

http://en.wikipedia.org/wiki/HTTP_tunnel#HTTP_CONNECT_tunneling
http://en.wikipedia.org/wiki/BOSH
http://xmpp.org/about-xmpp/technology-overview/bosh/

http://xmpp.org/extensions/xep-0124.html

aka long polling



*/
