/*
Copyright (C) 2021 Victor Fauth <victor@fauth.pro>

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see https://www.gnu.org/licenses/.
*/

// Package notifications provides helpers to send notifications
package notifications

import (
	"log"

	tb "gopkg.in/tucnak/telebot.v2"
)

// Structure storing the parameters required to send notifications with Telegram
type Telegram struct {
	Notifier *Notifier
}

// Telegram notifier specification
func TelegramInit(notifier *Notifier) Telegram {
	tg := Telegram{Notifier: notifier}
	tg.Notifier.Name = "telegram"
	tg.Notifier.Flags = map[string]*Flag{
		"token": {
			Long:      "tg-token",
			Short:     "",
			Help:      "Telegram notifier: Bot authentication token",
			ValueType: "string",
		},
		"userID": {
			Long:      "tg-user-id",
			Short:     "",
			Help:      "Telegram notifier: User ID of the user to notify",
			ValueType: "int",
		},
	}

	return tg
}

// Return the token
func (tg Telegram) token() string {
	return tg.Notifier.Flags["token"].Value.(string)
}

// Return the user ID
func (tg Telegram) userID() int {
	return tg.Notifier.Flags["userID"].Value.(int)
}

// Implement the sending of a notification to a Telegram user
func (tg Telegram) Send(message string) error {
	if tg.isConfigured() {
		log.Printf("Sending a Telegram notification to user %d", tg.userID())
		bot, _ := tb.NewBot(tb.Settings{Token: tg.token()})
		user := &tb.User{ID: tg.userID()}
		_, err := bot.Send(user, message)
		if err != nil {
			log.Printf("ERROR: Failed to send a Telegram notification to user %d, got: %s", tg.userID(), err)
		}

		return err
	}
	return nil
}

// Implement checking whether Telegram notifications are enabled
func (tg Telegram) isConfigured() bool {
	// Both the token and the user ID must be defined
	return tg.token() != "" && tg.userID() != 0
}
