/*
Copyright (C) 2021 Victor Fauth <victor@fauth.pro>

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see https://www.gnu.org/licenses/.
*/

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

	flag "github.com/spf13/pflag"
)

type Settings struct {
	url      string
	interval time.Duration
	quiet    bool
	watch    map[int]*Reward
}

type Project struct {
	name            string
	rewards         map[int]*Reward
	currency_symbol string
	initialized     bool
}

type Reward struct {
	id               int
	title            string
	title_with_price string
	price            int
	available        int
	limit            int
}

// Some global variables
var settings Settings
var project Project

func getProjectJSON() map[string]interface{} {
	// Download the project description page and extract its data
	res, err := http.Get(settings.url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("Could not get the project description, got HTTP response %d: \"%s\"", res.StatusCode, res.Status)
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

func getProjectData() {
	// Extract project data from the JSON
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

func parseArgs() {
	// Parse flags
	flag.IntSliceP("rewards", "r", []int{}, "Comma-separated list of unavailable limited rewards to watch, identified by their price in the project's original currency. If multiple limited rewards share the same price, all are watched. Ignored if --all is set.")
	flag.BoolP("all", "a", false, "If set, watch all unavailable limited rewards.")
	flag.DurationVarP(&settings.interval, "interval", "i", time.Minute, "Interval between checks")
	flag.BoolVarP(&settings.quiet, "quiet", "q", false, "Quiet mode.")
	help := *flag.BoolP("help", "h", false, "Display this help.")
	flag.CommandLine.SortFlags = false
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage: kickstarter-reward-notifier [OPTION] PROJECT_URL\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Print the help and exit
	if help {
		flag.Usage()
		os.Exit(0)
	}

	// Get and validate the project URL
	if len(flag.Args()) != 1 {
		fmt.Println("Invalid argument.")
		flag.Usage()
		os.Exit(1)
	}
	projectURL, err := url.ParseRequestURI(flag.Arg(0))
	if err != nil {
		log.Fatalf("Project URL not valid: %s", err)
	}
	projectURL.RawQuery = "" // Remove the query string
	if str.HasSuffix(projectURL.String(), "/description") {
		settings.url = projectURL.String()
	} else {
		settings.url = projectURL.String() + "/description"
	}
}

func registerWatchedRewards() {
	// List the rewards to watch
	if len(project.rewards) == 0 {
		fmt.Println("All of this project rewards are currently available.")
		os.Exit(0)
	}
	settings.watch = map[int]*Reward{}
	watchAll, _ := flag.CommandLine.GetBool("all")
	watchList, _ := flag.CommandLine.GetIntSlice("rewards")
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
	} else {
		askRewardsToWatch([]Reward{})
	}
}

func askRewardsToWatch(rewards []Reward) {
	i := 0
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

func findRewardsByPrice(price int) []int {
	// Return a slice containing the IDs of all rewards at the specified price
	rewards := []int{}
	for i, r := range project.rewards {
		if r.price == price {
			rewards = append(rewards, i)
		}
	}
	return rewards
}

func main() {
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
				fmt.Printf(`\n%s: %d/%d of reward "%s" available!\n`,
					time.Now().Format(time.Kitchen),
					r.available,
					r.limit,
					r.title_with_price)
			}
		}
		if !found && !settings.quiet {
			fmt.Print(".")
		}
	}
}
