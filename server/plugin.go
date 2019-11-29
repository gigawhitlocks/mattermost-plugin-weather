package main

import (
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"strings"
	"sync"

	"fmt"
	"github.com/gigawhitlocks/weather/nws"
	"io/ioutil"
	"path/filepath"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	botId string
}

type WeatherRequest struct {
	Zip  string
	Args *model.CommandArgs
}

func (p *Plugin) OnActivate() (err error) {
	p.botId, err = p.Helpers.EnsureBot(&model.Bot{
		Username:    "weather",
		DisplayName: "Weather",
		Description: "The Weather Bot",
	})

	if err != nil {
		return err
	}

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return err
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "weather.png"))
	if err != nil {
		p.API.LogError(fmt.Sprintf(err.Error(), "couldn't read profile image: %s"))
		return err
	}

	appErr := p.API.SetProfileImage(p.botId, profileImage)
	if appErr != nil {
		return appErr
	}

	// args.Command contains the full command string entered
	return p.API.RegisterCommand(&model.Command{
		Trigger:          "weather",
		DisplayName:      "Weather",
		Description:      "Gets the weather from the National Weather Service (US only) by zip code",
		AutoComplete:     true,
		AutoCompleteDesc: "/weather 78703 would return the weather for downtown Austin, TX.",
	})
}

func (p *Plugin) postWeather(req *WeatherRequest) {
	cc, err := nws.GetWeather(req.Zip)
	if err != nil {
		_ = p.API.SendEphemeralPost(req.Args.UserId, &model.Post{
			Message:   fmt.Sprintf("Couldn't get weather because %s", err.Error()),
			UserId:    req.Args.UserId,
			ChannelId: req.Args.ChannelId,
			ParentId:  req.Args.ParentId,
		})
		return
	}

	if cc == nil {
		_ = p.API.SendEphemeralPost(req.Args.UserId, &model.Post{
			Message:   fmt.Sprintf("No conditions found for %s", req.Zip),
			UserId:    req.Args.UserId,
			ChannelId: req.Args.ChannelId,
			ParentId:  req.Args.ParentId,
		})
		return
	}

	user, _ := p.API.GetUser(req.Args.UserId)
	if user == nil {
		p.API.LogError("Couldn't find user!")
	}

	output := fmt.Sprintf(
		"**Current conditions for %s from %s:**\n\n%s and %s°F degrees", cc.Name, cc.Station, cc.Conditions, cc.Temperature)

	if cc.PrecipitationLastHour > 0.009 {
		output = fmt.Sprintf("%s with %.01f inches of precipitation in the last hour.", output, cc.PrecipitationLastHour)
	} else {
		output = fmt.Sprintf("%s.", output)
	}

	if cc.WindGust > 0 {
		output = fmt.Sprintf("%s The wind gusted up to %.1f mph.", output, cc.WindGust)
	} else {
		output = fmt.Sprintf("%s The wind is calm.", output)
	}

	output = fmt.Sprintf("%s Requested by @%s", output, user.Username)

	p.API.CreatePost(&model.Post{
		Message:   output,
		UserId:    p.botId,
		ChannelId: req.Args.ChannelId,
		ParentId:  req.Args.ParentId,
	})

}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	zip := strings.TrimSpace(strings.TrimPrefix(args.Command, "/weather "))
	if len(zip) != 5 {
		return nil, model.NewAppError("weather plugin", "error: only 5 digit zip codes are supported input", nil, "input wasn't length 5", 400)
	}

	go p.postWeather(&WeatherRequest{
		Zip:  zip,
		Args: args,
	})

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Username:     "weather",
		IconURL:      "https://theknown.net/~ian/weather.png",
		Text:         fmt.Sprintf("Getting weather for %s", zip),
	}, nil
}
