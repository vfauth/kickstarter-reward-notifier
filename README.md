# kickstarter-reward-notifier

[![Go Reference](https://pkg.go.dev/badge/github.com/vfauth/kickstarter-reward-notifier.svg)](https://pkg.go.dev/github.com/vfauth/kickstarter-reward-notifier)

This is a simple Go script useful to watch a Kickstarter project and be notified when specified limited rewards are available.

## Usage

The script can interactively ask for the rewards to monitor, or you can use command-line arguments:

```text
Usage: kickstarter-reward-notifier [OPTION] PROJECT_URL
  -r, --rewards ints        Comma-separated list of unavailable limited rewards to watch, identified by their price in the project's original currency. If multiple limited rewards share the same price, all are watched. Ignored if --all is set
  -a, --all                 If set, watch all unavailable limited rewards
  -i, --interval duration   Interval between checks (default 1m0s)
  -q, --quiet               Quiet mode
  -t, --test-notification   Send a test notification at script start, fail if any configured notifier fails
  -h, --help                Display this help
      --tg-token string     Telegram notifier: Bot authentication token
      --tg-user-id int      Telegram notifier: User ID of the user to notify
```

## Notifications

The script supports sending notifications when a watched reward is available.
:warning: For now, a notification will be sent each time Kickstarter is polled and as long as the reward is available.

### Telegram

To send notifications with Telegram:

- create a bot using the BotFather ([see the official documentation](https://core.telegram.org/bots#6-botfather)) and pass the generated token using the `--tg-token` parameter
- get your user ID using by sending a message to [@userinfobot](https://telegram.me/userinfobot) and pass it using the `--tg-user-id` parameter
