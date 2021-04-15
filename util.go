package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	// Whether we are in development mode or not
	dev bool

	// Whether to display all the users you are following
	displayFollowing bool

	// Whether to display all the users following you
	displayFollowers bool
	
	// Whether to display users that are not following you back
	displayNotFollowingYouBack bool

	// Whether to display users that you are not following back
	displayYouDontFollowBack bool

	// Whether we want to launch the follow mode
	follow bool

	// Use this option to follow a list of users (set list with -followUserList)
	followUser bool

	// A comma seperated list of users to follow
	followUserList string

	// The max number of users to unfollow
	followLimit int

	// Auto approve following users
	forceFollow bool

	// Auto approve unfollowing users
	forceUnfollow bool

	// Whether we want an email to be sent when the script ends / crashes
	nomail bool

	// Max duration (second) to wait for between actions (must be higher than min duration)
	// The program will take a random duration between min and max after each ation
	maxSleepDuration int

	// Min duration (second) to wait for between actions (must be lower than max duration)
	// The program will take a random duration between min and max after each ation
	minSleepDuration int

	// Whether we want to launch the unfollow mode
	unfollow bool

	// The max number of users to unfollow
	unfollowLimit int

	// Whether we want to launch the unfollow mode
	unfollowUsers bool

	// A comma seperated list of users to unfollow
	unfollowUserList string

	// Acut
	run bool

	// Whether we want to have logging
	logs bool

	// Used to skip following, liking and commenting same user in this session
	noduplicate bool
)

// An image will be liked if the poster has more followers than likeLowerLimit, and less than likeUpperLimit
var likeLowerLimit int
var likeUpperLimit int

// A user will be followed if he has more followers than followLowerLimit, and less than followUpperLimit
// Needs to be a subset of the like interval
var followLowerLimit int
var followUpperLimit int

// An image will be commented if the poster has more followers than commentLowerLimit, and less than commentUpperLimit
// Needs to be a subset of the like interval
var commentLowerLimit int
var commentUpperLimit int

// Hashtags list. Do not put the '#' in the config file
var tagsList map[string]interface{}

// Limits for the current hashtag
var limits map[string]int

// Comments list
var commentsList []string

// Line is a struct to store one line of the report
type line struct {
	Tag, Action string
}

// Report that will be sent at the end of the script
var report map[line]int

var userBlacklist []string
var userWhitelist []string

// Counters that will be incremented while we like, comment, and follow
var numFollowed int
var numLiked int
var numCommented int

// Will hold the tag value
var tag string

// check will log.Fatal if err is an error
func check(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}

// Parses the options given to the script
func parseOptions() {
	flag.BoolVar(&run, "run", false, "Use this option to follow, like and comment")
	flag.BoolVar(&forceFollow, "forcefollow", false, "Use this option to auto approve following")
	flag.BoolVar(&followUser, "followUser", false, "Use this option to follow a list of users (set list with -followUserList)")
	flag.StringVar(&followUserList, "followUserList", "", "A comma seperated list of users to follow")
	flag.BoolVar(&follow, "follow", false, "Use this option to follow those who you don't follow back")
	flag.IntVar(&followLimit, "followlimit", 10, "Use this option to set the max users to follow (use with -follow)")
	flag.BoolVar(&forceUnfollow, "forceunfollow", false, "Use this option to auto approve unfollowing")
	flag.BoolVar(&unfollow, "unfollow", false, "Use this option to unfollow those who are not following back")
	flag.IntVar(&unfollowLimit, "unfollowlimit", 10, "Use this option to set the max users to unfollow (use with -unfollow)")
	flag.BoolVar(&unfollowUsers, "unfollowUsers", false, "Use this option to unfollow a list of users (set list with -unfollowUserList)")
	flag.StringVar(&unfollowUserList, "unfollowUserList", "", "A comma seperated list of users to unfollow")
	flag.BoolVar(&displayFollowing, "displayFollowing", false, "Whether to display all the users you are following")
	flag.BoolVar(&displayFollowers, "displayFollowers", false, "Whether to display all the users following you")
	flag.BoolVar(&displayNotFollowingYouBack, "displaynotfollowingyouback", false, "Use this option to display those who are not following back")
	flag.BoolVar(&displayYouDontFollowBack, "displayyoudontfollowback", false, "Use this option to display those you are not following back")
	flag.IntVar(&maxSleepDuration, "maxsleepduration", 35, "Use this option to set the max duration to wait between actions")
	flag.IntVar(&minSleepDuration, "minsleepduration", 10, "Use this option to set the min duration to wait between actions")
	flag.BoolVar(&nomail, "nomail", false, "Use this option to disable the email notifications")
	flag.BoolVar(&dev, "dev", false, "Use this option to use the script in development mode : nothing will be done for real")
	flag.BoolVar(&logs, "logs", false, "Use this option to enable the logfile")
	flag.BoolVar(&noduplicate, "noduplicate", false, "Use this option to skip following, liking and commenting same user in this session")

	flag.Parse()

	if minSleepDuration > maxSleepDuration {
		log.Fatalf("minsleepduration must be lower than maxsleepduration")
	}

	// -logs enables the log file
	if logs {
		// Opens a log file
		t := time.Now()
		logFile, err := os.OpenFile("instabot-"+t.Format("2006-01-02-15-04-05")+".log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		check(err)
		defer logFile.Close()

		// Duplicates the writer to stdout and logFile
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
	}
}

// Gets the conf in the config file
func getConfig() {
	folder := "config"
	if dev {
		folder = "local"
	}
	viper.SetConfigFile(folder + "/config.json")

	// Reads the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	// Check enviroment
	viper.SetEnvPrefix("instabot")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	// Confirms which config file is used
	log.Printf("Using config: %s\n\n", viper.ConfigFileUsed())

	likeLowerLimit = viper.GetInt("limits.like.min")
	likeUpperLimit = viper.GetInt("limits.like.max")

	followLowerLimit = viper.GetInt("limits.follow.min")
	followUpperLimit = viper.GetInt("limits.follow.max")

	commentLowerLimit = viper.GetInt("limits.comment.min")
	commentUpperLimit = viper.GetInt("limits.comment.max")

	tagsList = viper.GetStringMap("tags")

	commentsList = viper.GetStringSlice("comments")

	userBlacklist = viper.GetStringSlice("blacklist")
	userWhitelist = viper.GetStringSlice("whitelist")

	type Report struct {
		Tag, Action string
	}

	report = make(map[line]int)
}

// Sends an email. Check out the "mail" section of the "config.json" file.
func send(body string, success bool) {
	if !nomail {
		from := viper.GetString("user.mail.from")
		pass := viper.GetString("user.mail.password")
		to := viper.GetString("user.mail.to")

		status := func() string {
			if success {
				return "Success!"
			}
			return "Failure!"
		}()
		msg := "From: " + from + "\n" +
			"To: " + to + "\n" +
			"Subject:" + status + "  go-instabot\n\n" +
			body

		err := smtp.SendMail(viper.GetString("user.mail.smtp"),
			smtp.PlainAuth("", from, pass, viper.GetString("user.mail.server")),
			from, []string{to}, []byte(msg))

		if err != nil {
			log.Printf("smtp error: %s", err)
			return
		}

		log.Print("sent")
	}
}

// Retries the same function [function], a certain number of times (maxAttempts).
// It is exponential : the 1st time it will be (sleep), the 2nd time, (sleep) x 2, the 3rd time, (sleep) x 3, etc.
// If this function fails to recover after an error, it will send an email to the address in the config file.
func retry(maxAttempts int, sleep time.Duration, function func() error) (err error) {
	for currentAttempt := 0; currentAttempt < maxAttempts; currentAttempt++ {
		err = function()
		if err == nil {
			return
		}
		for i := 0; i <= currentAttempt; i++ {
			time.Sleep(sleep)
		}
		log.Println("Retrying after error:", err)
	}

	send(fmt.Sprintf("The script has stopped due to an unrecoverable error :\n%s", err), false)
	return fmt.Errorf("After %d attempts, last error: %s", maxAttempts, err)
}

// Builds the line for the report and prints it
func buildLine() {
	reportTag := ""
	for index, element := range report {
		if index.Tag == tag {
			reportTag += fmt.Sprintf("%s %d/%d - ", index.Action, element, limits[index.Action])
		}
	}
	// Prints the report line on the screen / in the log file
	if reportTag != "" {
		log.Println(strings.TrimSuffix(reportTag, " - "))
	}
}

// Builds the report prints it and sends it
func buildReport() {
	reportAsString := ""
	for index, element := range report {
		var times string
		if element == 1 {
			times = "time"
		} else {
			times = "times"
		}
		if index.Action == "like" {
			reportAsString += fmt.Sprintf("#%s has been liked %d %s\n", index.Tag, element, times)
		} else {
			reportAsString += fmt.Sprintf("#%s has been %sed %d %s\n", index.Tag, index.Action, element, times)
		}
	}

	// Displays the report on the screen / log file
	fmt.Println(reportAsString)

	// Sends the report to the email in the config file, if the option is enabled
	send(reportAsString, true)
}

func randomTimeSleep(min, max int) {
    randomSleepTime := rand.Intn(max - min + 1) + min
    time.Sleep(time.Duration(randomSleepTime) * time.Second)
}