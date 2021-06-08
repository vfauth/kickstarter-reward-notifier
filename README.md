# kickstarter-reward-notifier

This a simple Go script useful to watch a Kickstarter project and be notified when specified limited rewards are available.

## Usage

The script can interactively ask for the rewards to monitor, or you can use command-line arguments:

```bash
kickstarter-reward-notifier [OPTION] PROJECT_URL
  -r, --rewards ints        Comma-separated list of unavailable limited rewards to watch, identified by their price in the project's original currency.
                            If multiple limited rewards share the same price, all are watched. Ignored if --all is set.
  -a, --all                 If set, watch all unavailable limited rewards.
  -i, --interval duration   Interval between checks (default 1m0s)
  -h, --help                Display this help.
```

## Notifications

For now, it only outputs to stdout when watched rewards are available. Telegram notifications are in the work.
