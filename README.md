Here's a little IMAP-client program in Go.

* It is based on the [Go "imap" library](https://github.com/suntong/go-imap) that implements the IMAP client protocol 
* It is a more complete implementation that is capable to sync your online and off-line mails repeatedly without re-downloading the old messages, or to sync two mail folders from two different cloud mail accounts into a single mail box file.
* Just FTR, to send each message from within the mail box file as separated email to another (cloud) mail account, check out `formail` from the `procmail` package. I.e., you don't need to pay $$$ to sync messages between your different cloud mail accounts.
