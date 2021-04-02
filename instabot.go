package main

import (
	"strings"
	"time"

	"github.com/ahmdrz/goinsta/v2"
)

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
	if displayNotFollowingYouBack {
		instabot.displayUsersNotFollowingBack()
	} else if displayYouDontFollowBack {
		instabot.displayUsersYouDontFollowBack()
	} else if followUser {
		usersToFollow := strings.Split(followUserList, ",")
		for _, user := range usersToFollow {
			var instaUser *goinsta.User
			err := retry(10, 20*time.Second, func() (err error) {
				instaUser, err = instabot.Insta.Profiles.ByName(user)
				return
			})
			check(err)
			instabot.followUser(instaUser)
		}
	} else if unfollow {
		instabot.unfollowUsers()
	} else if run {
		// Loop through tags ; follows, likes, and comments, according to the config file
		instabot.loopTags()
	}
	instabot.updateConfig()
}
