package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/ahmdrz/goinsta/v2"
	"github.com/spf13/viper"
)

// Storing user in session
var checkedUser = make(map[string]bool)

// login will try to reload a previous session, and will create a new one if it can't
func login() {
	err := reloadSession()
	if err != nil {
		createAndSaveSession()
	}
}

// reloadSession will attempt to recover a previous session
func reloadSession() error {

	insta, err := goinsta.Import("./goinsta-session")
	if err != nil {
		return errors.New("Couldn't recover the session")
	}

	if insta != nil {
		instabot.Insta = insta
	}

	log.Println("Successfully logged in")
	return nil

}

// Logins and saves the session
func createAndSaveSession() {
	insta := goinsta.New(viper.GetString("user.instagram.username"), viper.GetString("user.instagram.password"))
	instabot.Insta = insta
	err := instabot.Insta.Login()
	check(err)

	err = instabot.Insta.Export("./goinsta-session")
	check(err)
	log.Println("Created and saved the session")
}

func getInput(text string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(text)
	input, err := reader.ReadString('\n')
	check(err)
	return strings.TrimSpace(input)
}

// Checks if the user is in the slice
func containsUser(slice []goinsta.User, user goinsta.User) bool {
	for _, currentUser := range slice {
		if currentUser.Username == user.Username {
			return true
		}
	}
	return false
}

// func getInputf(format string, args ...interface{}) string {
// 	reader := bufio.NewReader(os.Stdin)
// 	fmt.Printf(format, args...)
// 	input, err := reader.ReadString('\n')
// 	check(err)
// 	return strings.TrimSpace(input)
// }

// Same, with strings
func containsString(slice []string, user string) bool {
	for _, currentUser := range slice {
		if currentUser == user {
			return true
		}
	}
	return false
}

func (myInstabot MyInstabot) getDiffFollowersFollowing() []goinsta.User {
	following := myInstabot.Insta.Account.Following()
	followers := myInstabot.Insta.Account.Followers()

	var followerUsers []goinsta.User
	var followingUsers []goinsta.User
	var users []goinsta.User

	for following.Next() {
		followingUsers = append(followingUsers, following.Users...)
	}
	for followers.Next() {
		followerUsers = append(followerUsers, followers.Users...)
	}

	for _, user := range followingUsers {
		// Skip whitelisted users.
		if containsString(userWhitelist, user.Username) {
			continue
		}

		if !containsUser(followerUsers, user) {
			users = append(users, user)
		}
	}

	return users
}

func (myInstabot MyInstabot) displayUsersNotFollowingBack() {
	users := myInstabot.getDiffFollowersFollowing()

	var usernames []string
	fmt.Printf("The following users are not following you back:")
	for _, user := range users {
		usernames = append(usernames, user.Username)
	}
	sort.Strings(usernames)
	for _, user := range usernames {
		fmt.Printf("%s\n", user)
	}
}

func (myInstabot MyInstabot) unfollowUsers() {
	users := myInstabot.getDiffFollowersFollowing()

	if len(users) == 0 {
		fmt.Printf("All in syncing, ending now")
		return
	}

	fmt.Printf("\n%d users are not following you back!\n", len(users))
	fmt.Println("If you want to review these users, use -display")

	fmt.Printf("We are going to unfollow a max of %d users\n", unfollowLimit)

	fmt.Printf("We are going to unfollow these users\n")
	for i, user := range users {
		fmt.Printf("%d. %s\n", i, user.Username)
		if i == unfollowLimit - 1 {
			break
		}
		
	}
	
	if !forceUnfollow {
		if getInput("Start unfollowing? [yN]") != "y" {
			return
		}
	}

	for i, user := range users {
		userBlacklist = append(userBlacklist, user.Username)

		if !dev {
			fmt.Printf("Unfollowing %s\n", user.Username)
			err := user.Unfollow()
			if err != nil {
				fmt.Printf("In unfollowUsers: %s", err)
			}
		}
		randomTimeSleep(4, 20)

		if i == unfollowLimit - 1 {
			break
		}
	}
}

// Follows a user, if not following already
func (myInstabot MyInstabot) followUser(user *goinsta.User) {
	log.Printf("Following %s\n", user.Username)
	err := user.FriendShip()
	check(err)
	// If not following already
	if !user.Friendship.Following {
		if !dev {
			err := user.Follow()
			if err != nil {
				fmt.Printf("In followUser: %s", err)
			}
		}
		log.Println("Followed")
		numFollowed++
		report[line{tag, "follow"}]++
	} else {
		log.Println("Already following " + user.Username)
	}
}

func (myInstabot MyInstabot) loopTags() {
	for tag = range tagsList {
		limitsConf := viper.GetStringMap("tags." + tag)
		// Some converting
		limits = map[string]int{
			"follow":  int(limitsConf["follow"].(float64)),
			"like":    int(limitsConf["like"].(float64)),
			"comment": int(limitsConf["comment"].(float64)),
		}
		// What we did so far
		numFollowed = 0
		numLiked = 0
		numCommented = 0

		myInstabot.browse()
	}
	buildReport()
}

// Likes an image, if not liked already
func (myInstabot MyInstabot) likeImage(image goinsta.Item) {
	log.Println("Liking the picture")
	if !image.HasLiked {
		if !dev {
			err := image.Like()
			if err != nil {
				fmt.Printf("In unfollowUsers: %s", err)
			}
		}
		log.Println("Liked")
		numLiked++
		report[line{tag, "like"}]++
	} else {
		log.Println("Image already liked")
	}
}

func (myInstabot MyInstabot) browse() {
	var i = 0
	for numFollowed < limits["follow"] || numLiked < limits["like"] || numCommented < limits["comment"] {
		log.Println("Fetching the list of images for #" + tag + "\n")
		i++

		// Getting all the pictures we can on the first page
		// Instagram will return a 500 sometimes, so we will retry 10 times.
		// Check retry() for more info.
		var images *goinsta.FeedTag
		err := retry(10, 20*time.Second, func() (err error) {
			images, err = myInstabot.Insta.Feed.Tags(tag)
			return
		})
		check(err)

		myInstabot.goThrough(images)

		if viper.IsSet("limits.maxRetry") && i > viper.GetInt("limits.maxRetry") {
			log.Println("Currently not enough images for this tag to achieve goals")
			break
		}
	}
}

// Goes through all the images for a certain tag
func (myInstabot MyInstabot) goThrough(images *goinsta.FeedTag) {
	var i = 0

	// do for other too
	for _, image := range images.Images {
		// Exiting the loop if there is nothing left to do
		if numFollowed >= limits["follow"] && numLiked >= limits["like"] && numCommented >= limits["comment"] {
			break
		}

		// Skip our own images
		if image.User.Username == viper.GetString("user.instagram.username") {
			continue
		}

		// Check if we should fetch new images for tag
		if i >= limits["follow"] && i >= limits["like"] && i >= limits["comment"] {
			break
		}

		// Skip checked user if the flag is turned on
		if checkedUser[image.User.Username] && noduplicate {
			continue
		}

		// Getting the user info
		// Instagram will return a 500 sometimes, so we will retry 10 times.
		// Check retry() for more info.

		var userInfo *goinsta.User
		err := retry(10, 20*time.Second, func() (err error) {
			userInfo, err = myInstabot.Insta.Profiles.ByName(image.User.Username)
			return
		})
		check(err)

		followerCount := userInfo.FollowerCount

		buildLine()

		checkedUser[userInfo.Username] = true
		log.Println("Checking followers for " + userInfo.Username + " - for #" + tag)
		log.Printf("%s has %d followers\n", userInfo.Username, followerCount)
		i++

		// Will only follow and comment if we like the picture
		like := followerCount > likeLowerLimit && followerCount < likeUpperLimit && numLiked < limits["like"]
		follow := followerCount > followLowerLimit && followerCount < followUpperLimit && numFollowed < limits["follow"] && like
		comment := followerCount > commentLowerLimit && followerCount < commentUpperLimit && numCommented < limits["comment"] && like

		// Checking if we are already following current user and skipping if we do
		skip := false
		following := myInstabot.Insta.Account.Following()

		var followingUsers []goinsta.User
		for following.Next() {
			followingUsers = append(followingUsers, following.Users...)
		}

		for _, user := range followingUsers {
			if user.Username == userInfo.Username {
				skip = true
				break
			}
		}

		// Like, then comment/follow
		if !skip {
			if like {
				myInstabot.likeImage(image)
				if follow && !containsString(userBlacklist, userInfo.Username) {
					myInstabot.followUser(userInfo)
				}
				if comment {
					myInstabot.commentImage(image)
				}
			}
		}
		log.Printf("%s done\n\n", userInfo.Username)

		// This is to avoid the temporary ban by Instagram
		randomTimeSleep(15, 35)
	}
}

// Comments an image
func (myInstabot MyInstabot) commentImage(image goinsta.Item) {
	rand.Seed(time.Now().Unix())
	text := commentsList[rand.Intn(len(commentsList))]
	if !dev {
		comments := image.Comments
		if comments == nil {
			// monkey patching
			// we need to do that because https://github.com/ahmdrz/goinsta/pull/299 is not in goinsta/v2
			// I know, it's ugly
			newComments := goinsta.Comments{}
			rs := reflect.ValueOf(&newComments).Elem()
			rf := rs.FieldByName("item")
			rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
			item := reflect.New(reflect.TypeOf(image))
			item.Elem().Set(reflect.ValueOf(image))
			rf.Set(item)
			err := newComments.Add(text)
			if err != nil {
				fmt.Printf("In unfollowUsers: %s", err)
			}
			// end hack!
		} else {
			err := comments.Add(text)
			if err != nil {
				fmt.Printf("In unfollowUsers: %s", err)
			}
		}
	}
	log.Println("Commented " + text)
	numCommented++
	report[line{tag, "comment"}]++
}

func (myInstabot MyInstabot) updateConfig() {
	viper.Set("whitelist", userWhitelist)
	viper.Set("blacklist", userBlacklist)

	err := viper.WriteConfig()
	if err != nil {
		log.Println("Update config file error", err)
		return
	}

	log.Println("Config file updated")
}
