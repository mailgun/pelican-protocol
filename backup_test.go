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


*/
