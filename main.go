package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/google/go-querystring/query"
	"net/http"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "slack-to-email"
	app.Usage = "Extract email list(s) from specific slack channel"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "channel, c",
			Usage: "channel name",
		},
		cli.StringFlag{
			Name: "token, t",
			Usage: "slack web api token",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "show email list",
			Action:  listAction,
		},
	}

	app.Before = func(c *cli.Context) error {
		if c.GlobalString("channel") == "" {
			return errors.New("please set channel id")
		}

		if c.GlobalString("token") == "" {
			return errors.New("please set slack api token")
		}
		return nil
	}

	app.After = func(c *cli.Context) error {
		return nil
	}

	app.Run(os.Args)
}

type channelOptions struct {
	Token   string `url:"token"`
	Channel string `url:"channel"`
}

type channelInfo struct {
	Ok      bool `json:"ok"`
	Channel channel
}

type channel struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Members []string
}

func listAction(c *cli.Context) error {
	token := c.GlobalString("token")
	channelName := c.GlobalString("channel")
	opt := channelOptions{token, channelName}
	v, _ := query.Values(opt)
	resp, err := http.Get("https://slack.com/api/channels.info?" + v.Encode())
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	var info channelInfo
	json.NewDecoder(resp.Body).Decode(&info)

	emailChan := make(chan string, 100)

	var cnt int = 0
	for _, member := range info.Channel.Members {
		go getUserEmail(token, member, emailChan)
		cnt++
	}

	for {
		select {
		case mail := <-emailChan:
			fmt.Println(mail)
			cnt--
			if cnt == 0 {
				return nil
			}
		}
	}

	return nil
}

type userOptions struct {
	Token string `url:"token"`
	User  string `url:"user"`
}

type userInfo struct {
	Ok   bool `json:"ok"`
	User user
}

type user struct {
	Profile profile
}

type profile struct {
	RealName string `json:"real_name"`
	Email    string `json:"email"`
}

func getUserEmail(token string, userId string, emailChan chan string) error {
	opt := userOptions{token, userId}

	v, _ := query.Values(opt)
	resp, err := http.Get("https://slack.com/api/users.info?" + v.Encode())
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	var info userInfo
	encErr := json.NewDecoder(resp.Body).Decode(&info)

	if encErr != nil {
		panic(encErr)
	}

	emailChan <- info.User.Profile.Email

	return nil
}
