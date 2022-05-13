# Example Command

This is a simple Go script useful to watch a Kickstarter project and be notified when specified limited rewards are available.

```text
cd kickstarter-reward-notifier

go run main.go "https://www.kickstarter.com/projects/ankermake/ankermake-m5-3d-printer-5x-faster-printing-and-ai-camera" --tg-token "5194414408:AAGyiZ7DR4PmQRW1m9q6tROyh_6n90sefyI" --tg-user-id 5218885761 -i 0m10s -t
```
