protocol
------------

the computer knows that the public key of the server is okay.
how can the computer communicate that to the user?
"This isn't a site you've visited before. Create new login or browse anonymously?"

^^- why can't this be faked? it can, but a site cannot cause you to
click "create new login", and even if they can, we never present credentials
to the wrong person.

Key point: never give out prior credentials to a new site. Only ever give
out credentials to a site where that credential has been provided (and
then signed) before.


creating a new account vs connecting with an existing account.

"click through the warnings: not an option"

when the client doesn't recognize the server, it is creating a new account: how can the
client insist that this is a new account and not an earlier account that the mitm would
then compromise? by generating a new key pair and offering a newly generated public key
to the server. The server should provide a new account id in return.

pass-pictures: pick your sequence of pictures (9 choices, 9 choices)

is account id match required?

cases:
 if client is set to anon: then client always generates new key.
 ? but client should remember the key in case decide to upgrade
 ? upgrade from anon to full access should *not* be seamless: should require
    re-establishing the connection from the start, so that user doesn't
    accidentally start leaking info. ?

 new user (client gen new key) - client anon.
 new user (client gen new key) - client wants full access.
 returning user (prior client key) - client anon.
 returning user (prior client key) - full access.

advantages of rsa key over challenge response: challenge response can be recorded; rsa cannot be.




--> need to ask the user: which identity do you want to use to make this connection (anonymous / 

to save round trips, the client can validate the server and then present their public key.

