# BeepStarBot (Working Locally for now)
Copy of [NaaiveBot](https://github.com/101loop/naaive-bot), but this time written in [Go](https://golang.org/).

# How To Use it Locally
You need [Go](https://golang.org/), [ngrok](https://ngrok.com/download) and [curl](https://curl.se/docs/) as well installed in your system for this to work

- Fork/Clone this repo.
- Run `git clone link-to-repo.git` to clone this repo locally.
- Run `cd beepstarbot-go` - to go to the repo folder.
- Run `go get` to install dependencies.
- Create a new bot using [BotFather](https://t.me/botfather) and get your **BOT_TOKEN**.
- Copy .env.example file to .env and add BOT_TOKEN & SENTRY_DSN(this is optional, leave this empty if you want but
[Sentry](https://sentry.io) is a nice tool to log errors in production.)
- Run `go run main.go` (**Bot is now running**)
- Run `ngrok http 3000` (**First `cd` into the directory where you installed `ngrok` then run this command**)
- You'll see some public URLs after running the previous command.
- URLs will look like this `https://<some_code>.ngrok.io`(**notice https**), copy the URL which have `https`.
- Now run the following command
```shell script
curl -F "url=https://<some_code>.ngrok.io/"  https://api.telegram.org/bot<your_api_token>/setWebhook
```
This will let telegram know that our bot has to talk to this url whenever it receives any message.
- Now your bot ready to kick members.

**Note:** When you are done testing or if you restart ngrok then run the following command
```shell script
curl https://api.telegram.org/bot<your_api_token>/deleteWebhook
```
This will reset the webhook url.
# TODO: Add steps for deployment

# Acknowledgements
- [This](https://www.sohamkamani.com/golang/telegram-bot/) article helped a lot.
- [Golang bindings for the Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [Telegram API Docs](https://core.telegram.org/bots/api)

# Contributing
- Please file bugs and send pull requests to the [GitHub Repository](https://github.com/101Loop/beepstarbot-go) and [issue tracker](https://github.com/101Loop/beepstarbot-go/issues).
    - Don't be afraid to open half-finished PRs, and ask questions if something is unclear!
    - No contribution is too small! Please submit as many fixes for typos and grammar bloopers as you can!
    - Try to limit each pull request to one change only.
- For help and support please reach out to us on [Slack](https://101loop.slack.com).
