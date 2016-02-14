## Intro

Here's a little IMAP-client program in Go.

* It is based on the [Go "imap" library](https://github.com/suntong/go-imap) that implements the IMAP client protocol 
* It is a more complete implementation that is capable to sync your online and off-line mails repeatedly without re-downloading the old messages, or to sync two mail folders from two different cloud mail accounts into a single mail box file.
* Just FTR, to send each message from within the mail box file as separated email to another (cloud) mail account, check out `formail` from the `procmail` package. I.e., you don't need to pay $$$ to sync messages between your different cloud mail accounts.

## Usage

```
$ cloudmail 
Usage: cloudmail [global options] <verb> [verb options]

Global options:
        -v, --verbose Be verbose
        -h, --help    Show this help

Sub-commands (Verbs):
    list      List mailboxes
    fetch     Download mailbox
    sync      Sync cloud mail folder to existing mailbox file

Verbs:
    fetch:
          -f, --folder  Mail folder to fetch (*)
          -t, --trackid Track message Id
              --wy      Within years, only to fetch mails within this number of years
              --wm      Within months, ditto for months
              --wd      Within days, ditto for days
    list:
    sync:
          -f, --folder  Mail folder to fetch (*)
          -t, --trackid Track message Id
              --wy      Within years, only to fetch mails within this number of years
              --wm      Within months, ditto for months
              --wd      Within days, ditto for days
```

## Example

To list all Gmail labels:

    cloudmail list

To download all messages for one label:

    cloudmail fetch -f my-label

To download all while tracking each downloaded messages:

    cloudmail fetch -f my-label -t

And to restrict downloading to messages within recent two months:

    cloudmail fetch -f my-label -t --wm 2

Just for verification, the next example will redo the messages downloading, but this time will get recent six-month messages instead, to show that repeated downloading will not cause duplicated messages:

    cloudmail sync -f my-label -t --wm 6

## Configuration

To configure the program, create a file called `auth` that contains IMAP address, then the username and password (on one line each). Example setup has been provided in the `auth.example` file:

```
imap.gmail.com:993
userid
passwd
```

## Insight

- The downloaded mail box file will be named using the provided Gmail label (e.g., `my-label`) with the `.mbox` extension (e.g., `my-label.mbox`). To view it, use `mailx -f my-label.mbox`.
- If message-tracking is enabled, a message-tracking file, named using the provided Gmail label (e.g., `my-label`) with a `.yaml` extension (e.g., `my-label.yaml`), will be created/updated as well. It contains all the Message Ids from the downloaded messages. One Id for each message in sorted order. The Message Id is part of the RFC822 standard from mail header that uniquely identify an email message. They make re-downloading or even syncing between different cloud mail accounts possible. Thus, it is important that, once you've enabled message-tracking, you should always enable it in subsequent requests.

## Note

To access Gmail via IMAP, you need to [allow less secure apps to access your account](https://support.google.com/accounts/answer/6010255).



