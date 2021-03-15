package main

import "github.com/ahmdrz/goinsta/v2"

// MyInstabot is a wrapper around everything
type MyInstabot struct {
	Insta *goinsta.Instagram
}

var instabot MyInstabot

func main() {
	// Gets the command line options
	parseOptions()
	// Gets the config
	getConfig()
	// Tries to login
	login()
	if display {
		instabot.displayUsersNotFollowingBack()
	} else if unfollow {
		instabot.unfollowUsers()
	} else if run {
		// Loop through tags ; follows, likes, and comments, according to the config file
		instabot.loopTags()
	}
	instabot.updateConfig()
}
