# [IRChuu](https://26000.github.io/irchuu/) [![](https://goreportcard.com/badge/github.com/26000/irchuu)](https://goreportcard.com/report/github.com/26000/irchuu)
![IRChuu in WeeChat](https://26000.github.io/irchuu/images/shadow/Screenshot_20170319_092103.png)

A Telegram<->IRC transport.

## Features
- Relays messages between a Telegram (super)group and an IRC channel
- Lightweight, written in Go. Consumes only around 10MiB RAM!
- IRC authentication using SASL or NickServ
- (optional) Keeps log of the chat in a PostgreSQL database (those who recently joined the IRC channel can view history!)
- Preserves markup: bold in Telegram will remain bold in IRC
- All Telegram media types support; serves or uploads files so they are accessible in IRC
- All Telegram features like forwards, replies and edits are also supported
- Coloured nicknames in IRC
- (optional) Telegram group administrators can moderate the IRC channel and vice versa
- ...and this is not a complete list!

## Setup
### Installation
> **Note** IRChuu is written in Go and should work on all platforms supported by Go, including Linux, macOS and Windows. Nevertheless, I strongly recommend you using a Linux flavour to run IRChuu, or, as a last resort, macOS. IRChuu is only tested on Linux.

You need to install [golang](https://golang.org/doc/install), [git](https://git-scm.com/downloads) and to configure your `$GOPATH`. Just set `$GOPATH` environment variable to a writable directory and add `$GOPATH/bin` to your system `$PATH`.

After that, install IRChuu~:
```
$ go get github.com/26000/irchuu/cmd/irchuu
```

### Configuration
Run IRChuu~ for the first time and it will create a configuration file (you can also use `-data` and `-config` command-line arguments to specify a custom path):
```
$ irchuu
IRChuu! v0.6.0 (https://github.com/26000/irchuu)
2017/04/01 15:26:03 New configuration file was populated. Edit /home/26000/.config/irchuu.conf and run `irchuu` again!
```

Now edit the configuration file with your favourite editor (mine is `vim`, but I thought `nano` is more popular. Alternatively, you can just use a GUI editor like Kate):
```
$ nano ~/.config/irchuu.conf
```

The variables you *must* set are:
 - token, group in the `[telegram]` section
 - server, port, ssl, nick and channel in the `[irc]` section
 - probably serverpassword, password, sasl and chanpassword for IRC authentication

If you don't know where to get the Telegram token and groupname, refer to the next section.

Others are completely optional. The configuration file is well-documented, but if you have problems, feel free to open an issue on GitHub.

### Telegram bot setup
For IRChuu to work, you will need to create a Telegram bot as it works through the Telegram bot API. This is pretty simple:
1. Message [@botfather](http://t.me/botfather) inside Telegram. Send `/newbot` command.
2. It will ask some questions, answer all of them. You will have to think of a name and a nickname for your bot.
3. `@botfather` will send you a token. Insert it into your configuration file.
4. Type `/setprivacy` and choose your newly created bot nickname on the inline keyboard. Then choose **Disable**. This is important! If you forget to do it, messages from Telegram won't be relayed to IRC.
5. Optionally, type `/setuserpic` and upload a cute picure for your relay
6. Add your bot to your Telegram group (*Add members* and type the bot username there)
7. Launch IRChuu~ (just type `irchuu` in console once more)
8. Your bot will leave a message with the group id and quit. That's totally ok, just copy the id into your config file, stop IRChuu (hit <kbd>Ctrl</kbd>+<kbd>C</kbd> in the terminal where IRChuu~ is running)
9. All set, now just launch IRChuu for the third time and enjoy!

### IRC setup
This one is easier. You can just insert your server and channel addresses into your configuration file and choose a nickname. If that nickname is already taken, IRChuu will think of a new one. If you want to own that nickname so that nobody takes it, register it and enter the password in the configuration file. Refer to your server's NickServ focumentation for details.

## Usage
Just type `irchuu`.

## Contributing
Feel free to fork this repo and make PRs. If you encounter a bug, please open an issue — that also helps! I will also be happy if you give IRChuu a star on GitHub.

## Special thanks
To [zephy!](https://github.com/zephyyy) and [Kotobank!](https://kotobank.ch/) for support and motivation.
