# pelican-protocol
![diagram of pelican protocol in action](https://github.com/mailgun/pelican-protocol/blob/master/pelican.png "pelican-protocol-diagram")

Pelicans are ferocious eaters of fish. According to wikipedia, in ancient Egypt the pelican "was believed to possess the ability to prophesy safe passage in the underworld."[1] Pelicans are gregarious and nest colonially. Pairs are monogamous for a single season, but the pair bond extends only to the nesting area; mates are independent away from the nest.[2]


The Pelican Protocol provides a means of devouring phishing attempts. 

Advantages for website developers and operators
------------------------------------------------

Suppose you are a website owner who doesn't currently have SSL/TLS certificates and is running a good old plain HTTP site. You think TLS/x509 certificates are painful and costly, and you are right. By running the Pelican Server (reverse proxy) on your machine at port 443, you provide a zero-cost, high-security means of accessing your plain http website. And if decide to get certificates later, you can still keep running Pelican's additional anti-phishing protection.

Advantages for web browsers
---------------------------

Now suppose you are browsing the web using your favorite browser. The Pelican client (socks proxy) stores and manages your passwords (and SSH private keys) locally for you. The Pelican client provides you strong Man-in-the-Middle protection, much stronger than TLS/SSL, and you never had to mess with certificates or pay a dime.  You can still choose to browse any site anonymously, or you can authenticate strongly using your SSH private key. Backups are built-in so key recovery and moving keys to a new computer or phone is easy. Phishing is eliminated because once you've seen the real site, the Pelican client cannot be fooled by later seeing a look-alike site. This is the Trust-On-First-Use (TOFU) approach to security. Although not 100% foolproof, TOFU is still very effective because most phishing attacks are short-lived and occur well after your first visit to a site.

summary
-------

The Pelican Protocol provides means of doing user creation and authentication over an SSH-based protocol that tunnels http/s, and can be tunneled itself through https for full compatibility with existing practice. The Pelican protocol uses the SSH protocol for key-exchange and client-to-server port forwarding protocol; however Pelican but does not allow remote execution programs or shells. Pelican aims for strong usability by lay persons, and provides for portable and easy key management. A proxy for the client side does the key management and server identity checking, acting in the role of ssh-agent and the ssh client. The server provides a reverse proxy so you don't have change the webserver's configuration or buy certificates. The Pelican server provides an auto-login mechanism, easy key rotation and backup, and can be configured to allow logins for users only from known hosts. Two factor authentication via sms/text is available.

question
----------
Q: How does Pelican Client know that the Pelican Server is running, and so the http/s URL should be accessed over the SSH tunnel established outbound to port 443?

A: An example implementation of protcol discernment idea (telling HTTPS and SSH apart) is here (https://github.com/JamesDunne/sslmux). If Pelican Protocol is detected, the client can remember this and insist on Pelican in the future (to detect/deflect downgrade attacks).

question
--------
Q: How does the Pelican protocol acheive backwards compatibility with existing web clients and servers?

A: For each URL requested, the Pelican Client first attempts to connect to that host on port 443 using the Pelican protocol. In doing so, it determines whether the host is running HTTPS or the Pelican Protocol on port 443.  If the Pelican server is found, then the server is authenticated and the client's http or https traffic is tunneled to the webserver via an SSH secured forward-only channel. If the Pelican server is not present, then a normal HTTPS connection is used. This means that a user can run the Pelican client transparently alongside their web browser, without requiring that websites have the Pelican server installed. Likewise, a website operator can run the Pelican server without requiring that all clients run the Pelican client.

In the future, browsers could internalize the Pelican client to provide the protocol without needing the separate installatin of the Pelican client (and configuration of the socks proxy in the browser settings) on the browser's host.  In the future, popular webservers could incorporate the Pelican server protocol to make it easy to activate.  Once Pelican-enabled clients and servers become sufficiently popular and prevalent, websites could then start to choose to provide a Phishing-free, Pelican-only option (e.g. to avoid active MITM downgrade attacks like SSL stripping).

Future possibility: If the server is running HTTPS, then Pelican client could check to see if the server supports the Pelican protocol inside HTTPS to further protect against phishing and SSL stripping. This requires that, during installation of the Pelican server, the installer identify the location of the TLS/SSL certificates so that the Pelican server can accept the HTTPS connection directly. Since this is not needed for prototyping, we omit this feature from the first round of Pelican protocol implementation.

question
----------
Q: Is the server's hostkey bound to a particular domain name suffix or IP address, as in TLS certificates?

A: Nope. The private RSA key that identifies the server can be moved, backed-up, and restored onto a different IP address. Does this mean that if my private keys are stolen, then my site can be impersonated? Absolutely. Protect your keys with a strong passphrase, and use the auto-backup so you don't loose them.



[1]  Hart, George (2005). The Routledge Dictionary Of Egyptian Gods And Goddesses. Routledge Dictionaries. Abingdon, United Kingdom: Routledge. p. 125. ISBN 978-0-415-34495-1. Cite 99 of http://en.wikipedia.org/wiki/Pelican.

[2] http://en.wikipedia.org/wiki/Pelican

