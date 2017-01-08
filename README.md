# IRChuu
A Telegram<->IRC transport.

## Roadmap
- [x] Basics: config reading

- [x] Joining IRC channels: join, wait for an invite, join with a password
- [x] Joining Telegram group: leave wrong groups
- [x] NickServ login for IRC
- [x] SASL login for IRC

- [x] Relaying IRC messages and notices to Telegram
- [x] Relaying Telegram messages to IRC
- [x] Coloured nicknames
- [x] Preserve formatted text
- [x] /me command support from IRC
- [x] Replies, forwards support
- [x] Files: showing them as links in IRC, either uploading to a pomf-like hosting or serving with a built-in server
  - [x] Built-in server
  - [x] Pomf clones support
  - [ ] [Komf](https://github.com/koto-bank/komf) support
- [x] Moderation of the Telegram group from IRC
- [x] Moderation of the IRC channel from Telegram
- [x] Invitations to IRC from Telegram
- [x] Relaying IRC joins, parts, kicks and mode changes (configurable)

- [x] Logging to a DB
- [x] Showing history in IRC on query

- [x] Showing moderator list from IRC in TG
- [x] Showing moderator list from TG in IRC
- [x] Ability to contact IRC bots
- [x] Edits support
- [x] Breaking long Telegram messages into several IRC messages
- [x] Auto-reconnecting if internet lost
- [x] Autorejoin when kicked from IRC
- [x] Pause the message queue when kicked

- [ ] WeeChat plugin
- [ ] Unit tests
- [x] Automatically check for updates

- [ ] Docs
- [ ] A nice README.md file

