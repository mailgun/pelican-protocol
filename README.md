# pelican-protocol: Do-It-Yourself-Trust
![diagram of pelican protocol in action](https://github.com/mailgun/pelican-protocol/blob/master/pelican3.png "pelican-protocol-diagram")

Status
=====

(DO NOT USE YET!)

Still pre-alpha, lots of bugs and it is incomplete. Think of the description below as a proposal for the future.


overview
---------

The Pelican Protocol provides a means of devouring phishing attempts. It applies
SSH technology to the web, bringing the technology developed for secure 
console login to everyone. It authenticates both web users and web servers.
Every website you visit using the Pelican Protcol knows you by a unique key,
and it is impossible to mix up credentials and sites. Instead of 100s of
passwords to create and remember, there is a single strong pass phrase that
is entered once at machine start time.

This is a greenfield project. We deliberately ignore the prior
work of TLS, SSL, and certificate authorities. 

We propose a decentalized, do-it-yourself trust model. Our design offers
a sane user experience for both users of web browsers and for
web application developers.


Advantages for website developers and operators
------------------------------------------------

Suppose you are a website owner who doesn't currently have SSL/TLS certificates and is running a good old plain HTTP site. You think TLS/x509 certificates are painful and costly, and you are right. By running the Pelican Server (reverse proxy) on your machine, you provide a zero-cost, high-security means of accessing your plain http website. And if you decide to get certificates later, you can still keep running Pelican's additional anti-phishing protection for stronger anti-phishing.

Advantages for web browsers
---------------------------

Now suppose you are browsing the web using your favorite browser. The Pelican client (socks proxy) stores and manages your passwords (and SSH private keys) locally for you. The Pelican client provides you strong Man-in-the-Middle protection, and you never had to mess with certificates or pay a dime.  You can still choose to browse any site anonymously, or you can authenticate strongly using your SSH private key. Backups are built-in so key recovery and moving keys to a new computer or phone is easy. Phishing is eliminated because once you've seen the real site, the Pelican client cannot be fooled by later seeing a look-alike site. This is the Trust-On-First-Use (TOFU) approach to security. TOFU is effective because most phishing attacks are short-lived and occur well after your first visit to a site. WIFI hotspot SSL stripping is a thing of the past with Pelican on your side.

And even if an attacker happens to Man-In-The-Middle your first visit to a website, you are still protected: Pelican never re-uses credentials for different servers. The worst that can happen is that you create an account on a fake phishing site. However those credentials will never ever be used to authenticate against the real website. The worst that happens is that you learn later that somebody Man-in-the-middled your previous account creation on a fake website. Evesdroppers never learn any re-usable passwords.

summary
-------

The Pelican Protocol provides means of doing user creation and authentication over an SSH-based protocol that tunnels http.  The Pelican protocol uses the SSH protocol for key-exchange and client-to-server port forwarding protocol; however Pelican but does not allow remote execution of programs or shells. Pelican aims for strong usability by everyone, and provides for portable and easy key management. A proxy for the client side does the key management and server identity checking, acting in the role of ssh agent and client. The server provides a reverse proxy so you don't have to change your webserver's configuration. The Pelican server provides an auto-login mechanism, easy key rotation and server key backup, and can be configured to allow logins for users only from known hosts. Two factor authentication via sms/text is available. Deploy Pelican today and devour phish!

questions
----------
Q: How does Pelican Client know that the Pelican Server is running or not, if both Pelican and http are travelling to server port 80? Does this leave downgrade attacks a possiblity by an active MITM SSL-stripping style attack?

A: First, we can't solve all problems at once. An active MITM attack via protocol stripping is not the threat addressed here. The problem that Pelican solves is phishing via email links that take you to impersonated websites.  The reverse proxy on the server can readily distinguish Pelican traffic from non-Pelican http requests, and the client can recognize if Pelican is in use our not. Once Pelican Protocol is detected for a given server, the client can remember this and insist on Pelican in the future (to detect/deflect downgrade attacks).

question
--------
Q: How does the Pelican protocol acheive backwards compatibility with existing web clients and servers?

A: For each URL requested, the Pelican Client first attempts to connect to that host using the Pelican protocol. In doing so, it determines whether the host is running HTTP or the Pelican Protocol.  If the Pelican server is found, then the server is authenticated and the client's http or https traffic is tunneled to the webserver via an SSH secured forward-only channel. If the Pelican server is not present, then a normal HTTP connection is used. This means that a user can run the Pelican client transparently alongside their web browser, without requiring that websites have the Pelican server installed. Likewise, a website operator can run the Pelican server without requiring that all clients run the Pelican client.

In the future, browsers could internalize the Pelican client to provide the protocol without needing the separate installation of the Pelican client proxy.  In the future, popular webservers could incorporate the Pelican server protocol to make it even easer to deploy.  Once Pelican-enabled clients and servers become sufficiently popular and prevalent, websites could then start to choose to provide a Phishing-free, Pelican-only option. For now, these proxies are used to provide the protocol with one-time only installation.


technical question
----------
Q: Is the server's hostkey bound to a particular domain name suffix or IP address, as in TLS certificates?

A: Nope. The private RSA key that identifies the server can be moved, backed-up, and restored onto a different IP address. Does this mean that if my private keys are stolen, then my site can be impersonated? Absolutely. Protect your keys with a strong passphrase, and use the auto-backup so you don't loose them.


