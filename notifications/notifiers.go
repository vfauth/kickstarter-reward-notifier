/*
Copyright (C) 2021 Victor Fauth <victor@fauth.pro>

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see https://www.gnu.org/licenses/.
*/

// Package notifications provides helpers to send notifications
package notifications

import (
	"errors"
	"fmt"
	"reflect"
)

// Structure describing a notifier
type Notifier struct {
	Name  string           // Notifier name
	Flags map[string]*Flag // Slice of all flags used to pass parameters
}

// Structure listing all notifiers
type AllNotifiers struct {
	notifiers []*Notifier // Pointers to every embedded Notifier struct
	Telegram  Telegram    // Each notifier type embeds a Notifier struct
}

// Global variable containing each notifier
var allNotifiers AllNotifiers

// Structure describing a flag to pass notifiers parameters in the CLI
type Flag struct {
	Long      string      // Long name of the flag, required
	Short     string      // Short name
	Help      string      // Help message, required
	ValueType string      // Type of the value: "string", "int" or "bool" are supported
	Value     interface{} // Interface to hold the flag value
}

// Initialize all the notifiers and return them
func InitNotifiers() []*Notifier {
	allNotifiers.notifiers = make([]*Notifier, reflect.ValueOf(allNotifiers).NumField()-1)
	for i := range allNotifiers.notifiers {
		allNotifiers.notifiers[i] = &Notifier{}
	}
	allNotifiers.Telegram = TelegramInit(allNotifiers.notifiers[0])
	return allNotifiers.notifiers
}

// Send a notification using all configured notifiers
func SendNotification(message string) error {
	for _, notifier := range allNotifiers.notifiers {
		err := error(nil)
		switch notifier.Name {
		case "telegram":
			err = allNotifiers.Telegram.Send(message)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Send a notification using all configured notifiers
func (notifier Notifier) IsConfigured() bool {
	switch notifier.Name {
	case "telegram":
		return allNotifiers.Telegram.isConfigured()
	default:
		return false
	}
}

// Test all enabled notifiers
func TestNotifiers() error {
	// Count the configured notifiers
	enabled := 0
	for _, n := range allNotifiers.notifiers {
		if n.IsConfigured() {
			enabled++
		}
	}
	if enabled == 0 {
		return errors.New("no notifier has been configured")
	}

	message := "This is a test notification"
	failures := []error(nil)
	for _, notifier := range allNotifiers.notifiers {
		err := error(nil)
		switch notifier.Name {
		case "telegram":
			err = allNotifiers.Telegram.Send(message)
		}
		if err != nil {
			failures = append(failures, err)
		}
	}
	if len(failures) != 0 {
		errorMessage := fmt.Sprintf("Failure while testing notifiers: %d/%d notifiers returned an error.\n", len(failures), enabled)
		for _, f := range failures {
			errorMessage += f.Error() + "\n"
		}
		return errors.New(errorMessage)
	} else {
		return nil
	}
}
