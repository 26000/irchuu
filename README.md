# IRChuu
A Telegram<->IRC transport.

## Roadmap
- [x] Basics: config reading

- [x] Joining IRC channels: join, wait for an invite, join with a password
- [x] Joining Telegram group: leave wrong groups
- [x] NickServ login for IRC
- [x] SASL login for IRC

- [x] Relaying IRC messages to Telegram
- [x] Relaying Telegram messages to IRC
- [x] Coloured nicknames
- [x] Preserve formatted text
- [x] /me command support from IRC
- [x] Replies, forwards support
- [x] Documents: serving them as links in IRC
- [ ] Pictures: serving and Imgur uploads
- [x] Moderation of the Telegram group from IRC
- [x] Moderation of the IRC channel from Telegram
- [x] Invitations to IRC from Telegram

- [x] Logging to a DB
- [x] Showing history in IRC on query

- [x] Showing moderator list from IRC in TG
- [x] Showing moderator list from TG in IRC
- [x] Ability to contact IRC bots
- [x] Edits support
- [x] Breaking long Telegram messages into several IRC messages
- [x] Auto-reconnecting if internet lost

- [ ] WeeChat plugin
- [ ] Unit tests
- [ ] Automatic check for updates
- [ ] Reorganize code, split some files into several

- [ ] PM support

Bugs:
- [ ] Find a better alternative to polling the server for names
