/*
Copyright (C) 2021 Victor Fauth <victor@fauth.pro>

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see https://www.gnu.org/licenses/.
*/

// Get notified when limited rewards on Kickstarter are available
package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	str "strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/vfauth/kickstarter-reward-notifier/notifications"

	"github.com/spf13/pflag"
)

// Structure storing the script parameters
type Settings struct {
	url       string                    // Project description URL
	interval  time.Duration             // Interval between polling
	quiet     bool                      // Quiet mode
	watch     map[int]*Reward           // Map of rewards to watch, indexed by their ID
	notifiers []*notifications.Notifier // Slice of all the available notifiers
}

// Structure storing the project details
type Project struct {
	name            string          // Project name
	rewards         map[int]*Reward // Map of all limited rewards, indexed by their ID
	currency_symbol string          // The symbol representing the project currency
	initialized     bool            // Whether that project immutable data has already been obtained
}

// Structure storing the details about a specific reward
type Reward struct {
	id               int    // Kickstarter ID of this reward
	title            string // Reward name
	title_with_price string // Reward name including its price
	price            int    // Reward price in the project original currency
	available        int    // Remaining number of this reward
	limit            int    // Total quantity of this reward
}

// Global Settings structure containing the script parameters
var settings Settings

// Global Project structure containing the project details
var project Project

// Obtain the data about the project and store it in the `project` global variable
func getProjectData() {
	data := getProjectJSON()
	// The first time, get immutable data
	if !project.initialized {
		project.name = data["name"].(string)
		project.currency_symbol = data["currency_symbol"].(string)
		project.rewards = map[int]*Reward{}
		for _, r := range data["rewards"].([]interface{}) {
			reward := r.(map[string]interface{})
			_, limited := reward["limit"]
			if limited && reward["remaining"].(float64) == 0 {
				id := int(reward["id"].(float64))
				project.rewards[id] = &Reward{
					title:            reward["title"].(string),
					title_with_price: reward["title_for_backing_tier"].(string),
					id:               id,
					price:            int(reward["minimum"].(float64)),
				}
			}
		}
		project.initialized = true
	}
	// Get mutable data
	for _, r := range data["rewards"].([]interface{}) {
		reward := r.(map[string]interface{})
		_, limited := reward["limit"]
		if limited && reward["remaining"].(float64) == 0 {
			id := int(reward["id"].(float64))
			project.rewards[id].available = int(reward["remaining"].(float64))
			project.rewards[id].limit = int(reward["limit"].(float64))
		}
	}
}

// Download the project description page and return the unmarshalled JSON object containing the project data
func getProjectJSON() map[string]interface{} {
	res, err := http.Get(settings.url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf(
			"Could not get the project description, got HTTP response %d: \"%s\"",
			res.StatusCode,
			res.Status)
	}

	// Load the HTML document
	description, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Parse the HTML and extract the JSON describing the project
	jsonRegexp := regexp.MustCompile(`window\.current_project\s*=\s*"(\{.*\})"`)
	var projectDetails map[string]interface{}
	description.Find("script").EachWithBreak(func(i int, s *goquery.Selection) bool {
		match := jsonRegexp.FindStringSubmatch(s.Text())
		if match != nil {
			json.Unmarshal([]byte(html.UnescapeString(match[1])), &projectDetails)
			// Exit the loop
			return false
		}
		return true
	})
	return projectDetails
}

// Parse flags and store the results in the `settings` global variable
func parseArgs() {
	// Define flags
	pflag.IntSliceP("rewards", "r", []int{}, "Comma-separated list of unavailable limited rewards to watch, identified by their price in the project's original currency. If multiple limited rewards share the same price, all are watched. Ignored if --all is set")
	pflag.BoolP("all", "a", false, "If set, watch all unavailable limited rewards")
	pflag.DurationVarP(&settings.interval, "interval", "i", time.Minute, "Interval between checks")
	pflag.BoolVarP(&settings.quiet, "quiet", "q", false, "Quiet mode")
	notificationTest := pflag.BoolP("test-notification", "t", false, "Send a test notification at script start, fail if any configured notifier fails")
	help := pflag.BoolP("help", "h", false, "Display this help")

	// Setup the notifiers flags
	for _, notifier := range settings.notifiers {
		for _, flag := range notifier.Flags {
			switch flag.ValueType {
			case "string":
				pflag.StringP(flag.Long, "", "", flag.Help)
			case "int":
				pflag.IntP(flag.Long, "", 0, flag.Help)
			case "bool":
				pflag.BoolP(flag.Long, "", false, flag.Help)
			default:
				log.Fatalf(
					`Error in notifier "%s": "%s" is not supported as a notifier flag type\n`,
					notifier.Name,
					flag.ValueType)
			}
		}
	}

	// Configure and parse the flags
	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		fmt.Printf("Usage: kickstarter-reward-notifier [OPTION] PROJECT_URL\n")
		pflag.PrintDefaults()
	}
	pflag.Parse()

	// Print the help and exit
	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	// Get the notifiers flags values
	for _, notifier := range settings.notifiers {
		for _, flag := range notifier.Flags {
			switch flag.ValueType {
			case "string":
				flag.Value, _ = pflag.CommandLine.GetString(flag.Long)
			case "int":
				flag.Value, _ = pflag.CommandLine.GetInt(flag.Long)
			case "bool":
				flag.Value, _ = pflag.CommandLine.GetBool(flag.Long)
			}
		}
	}

	// Test the notifiers
	if *notificationTest {
		fmt.Println("Testing the notifications...")
		err := notifications.TestNotifiers()
		if err != nil {
			fmt.Printf("Failure during notification test: %s", err)
			os.Exit(1)
		} else {
			fmt.Println("All configured notifiers passed the test.")
		}
	}

	// Get and validate the project URL
	if len(pflag.Args()) != 1 {
		pflag.Usage()
		fmt.Printf("Invalid argument: there must be a single URL passed as parameter.\n")
		os.Exit(1)
	}
	projectURL, err := url.ParseRequestURI(pflag.Arg(0))
	if err != nil {
		fmt.Printf("Project URL not valid: %s", err)
		os.Exit(1)
	}
	projectURL.RawQuery = "" // Remove the query string
	if str.HasSuffix(projectURL.String(), "/description") {
		settings.url = projectURL.String()
	} else {
		settings.url = projectURL.String() + "/description"
	}
}

// Determine the rewards to watch
func registerWatchedRewards() {
	if len(project.rewards) == 0 {
		fmt.Println("All of this project rewards are currently available.")
		os.Exit(0)
	}
	settings.watch = map[int]*Reward{}
	watchAll, _ := pflag.CommandLine.GetBool("all")
	watchList, _ := pflag.CommandLine.GetIntSlice("rewards")
	if watchAll {
		settings.watch = project.rewards
	} else if len(watchList) != 0 {
		for _, price := range watchList {
			r := findRewardsByPrice(price)
			if len(r) == 0 {
				fmt.Printf("There is no limited and unavailable reward priced at %d%s, ignoring.\n", price, project.currency_symbol)
			} else {
				for i := range r {
					settings.watch[i] = project.rewards[i]
				}
			}
		}
	}

	// Prompt the user if no reward was specified
	if len(settings.watch) == 0 {
		askRewardsToWatch([]Reward{})
	}

	// Display list of watched rewards
	summary := fmt.Sprintf("%d rewards watched:\n", len(settings.watch))
	for _, w := range settings.watch {
		summary += fmt.Sprintf("- %s\n", w.title_with_price)
	}
	fmt.Print(summary)
}

// Prompt the user to interactively choose which limited rewards should be watched
func askRewardsToWatch(rewards []Reward) {
	i := 0
	// Map the prompt index to the reward ID
	rewardIndex := map[int]*Reward{}
	choices := []string{}
	for _, reward := range project.rewards {
		choices = append(choices, fmt.Sprintf("%s (%d backers)", reward.title_with_price, reward.limit))
		rewardIndex[i] = reward
		i++
	}
	prompt := &survey.MultiSelect{
		Message:  "Please select the rewards to watch:",
		Options:  choices,
		PageSize: 100,
	}
	selection := []int{}
	survey.AskOne(prompt, &selection, survey.WithValidator(survey.Required))
	for _, i := range selection {
		id := rewardIndex[i].id
		settings.watch[id] = rewardIndex[i]
	}
}

// Return a slice containing the IDs of all rewards at the specified price
func findRewardsByPrice(price int) []int {
	rewards := []int{}
	for i, r := range project.rewards {
		if r.price == price {
			rewards = append(rewards, i)
		}
	}
	return rewards
}

//  Script entrypoint
func main() {
	settings.notifiers = notifications.InitNotifiers()
	parseArgs()
	// Get the project data and rewards list
	getProjectData()
	registerWatchedRewards()
	for {
		time.Sleep(settings.interval)
		getProjectData()
		found := false
		for _, r := range settings.watch {
			if r.available > 0 {
				found = true
				message := fmt.Sprintf(`%d/%d of reward "%s" available!`,
					r.available,
					r.limit,
					r.title_with_price)
				notifMessage := fmt.Sprintf(`Alert about Kickstarter project "%s": %s`, project.name, message)
				log.Printf(`\n%s\n`, message)
				notifications.SendNotification(notifMessage)
			}
		}
		if !found && !settings.quiet {
			fmt.Print(".")
		}
	}
}
