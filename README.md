# pelican-protocol
![diagram of pelican protocol in action](https://github.com/mailgun/pelican-protocol/blob/master/pelican.png "pelican-protocol-diagram")

Pelicans are ferocious eaters of fish. According to wikipedia, in ancient Egypt the pelican "was believed to possess the ability to prophesy safe passage in the underworld."[1] Pelicans are gregarious and nest colonially. Pairs are monogamous for a single season, but the pair bond extends only to the nesting area; mates are independent away from the nest.[2]


The Pelican Protocol provides a means of devouring phishing attempts. 

Advantages for website developers and operators
------------------------------------------------

Suppose you are a website owner who doesn't currently have SSL/TLS certificates and is running a good old plain HTTP site. You think TLS/x509 certificates are painful and costly, and you are right. By running the Pelican Server (reverse proxy) on your machine at port 443, you provide a zero-cost, high-security means of accessing your plain http website. And if decide to get certificates later, you can still keep running Pelican's additional anti-phishing protection.

Advantages for web browsers
---------------------------

Now suppose you are browsing the web using your favorite browser. The Pelican client (socks proxy) stores and manages your passwords (and SSH private keys) locally for you. The Pelican client provides you strong Man-in-the-Middle protection, much stronger than TLS/SSL, and you never had to mess with certificates or pay a dime.  You can still choose to browse any site anonymously, or you can authenticate strongly using your SSH private key. Backups are built-in so key recovery and moving keys to a new computer or phone is easy. Phishing is eliminated because once you've seen the real site, the Pelican client cannot be fooled by later seeing a look-alike site.

summary
-------

The Pelican Protocol provides means of doing user creation and authentication over an SSH based protocol that tunnels http/https. It aims for strong usability by lay persons, and provides for portable and easy key management. A proxy for the client side does the key management and server identity checking, acting in the role of ssh-agent and the ssh client. On the server side, a reverse proxy and a secure login shell to implement the server side of the protocol are provided.



[1]  Hart, George (2005). The Routledge Dictionary Of Egyptian Gods And Goddesses. Routledge Dictionaries. Abingdon, United Kingdom: Routledge. p. 125. ISBN 978-0-415-34495-1. Cite 99 of http://en.wikipedia.org/wiki/Pelican.

[2] http://en.wikipedia.org/wiki/Pelican

