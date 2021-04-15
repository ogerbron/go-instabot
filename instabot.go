package main

import (
	"fmt"
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
	} else if displayFollowers {
		instabot.displayFollowers()
	} else if displayFollowing {
		instabot.displayFollowing()
	} else if followUser {
		usersToFollow := strings.Split(followUserList, ",")
		if len(usersToFollow) == 0 {
			fmt.Printf("The list of users to follow is empty.\n")
			return
		} else {
			for _, user := range usersToFollow {
				var instaUser *goinsta.User
				err := retry(10, 20*time.Second, func() (err error) {
					instaUser, err = instabot.Insta.Profiles.ByName(user)
					return
				})
				check(err)
				instabot.followUser(instaUser)
			}
		}
	} else if unfollowUsers {
		usersToUnfollow := strings.Split(unfollowUserList, ",")
		if len(usersToUnfollow) == 0 {
			fmt.Printf("The list of users to unfollow is empty.\n")
			return
		} else {
			for _, user := range usersToUnfollow {
				var instaUser *goinsta.User
				err := retry(10, 20*time.Second, func() (err error) {
					instaUser, err = instabot.Insta.Profiles.ByName(user)
					return
				})
				check(err)
				instabot.unfollowUser(instaUser)
			}
		}
	} else if unfollow {
		instabot.unfollowUsers()
	} else if run {
		// Loop through tags ; follows, likes, and comments, according to the config file
		instabot.loopTags()
	}
	instabot.updateConfig()
}
