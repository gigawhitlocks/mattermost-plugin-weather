package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gigawhitlocks/weather/climacell"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	botId        string
	profileImage []byte
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

	p.profileImage, err = ioutil.ReadFile(filepath.Join(bundlePath, "assets", "weather.png"))
	if err != nil {
		p.API.LogError(fmt.Sprintf(err.Error(), "couldn't read profile image: %s"))
		return err
	}

	appErr := p.API.SetProfileImage(p.botId, p.profileImage)
	if appErr != nil {
		return appErr
	}

	// args.Command contains the full command string entered
	return p.API.RegisterCommand(&model.Command{
		Trigger:          "weather",
		DisplayName:      "Weather",
		Description:      "Gets the weather",
		AutoComplete:     true,
		AutoCompleteDesc: "Search for a location like \"1600 Pennsylvania Ave\", \"Miami\", \"Purdue University\", or even \"The Statue of Liberty\".",
	})
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	w.Write(p.profileImage)
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	input := strings.TrimSpace(strings.TrimPrefix(args.Command, "/weather "))
	//iconURL := fmt.Sprintf("%s/plugins/%s?weather.png", *p.API.GetConfig().ServiceSettings.SiteURL, manifest.ID)
	output, err := climacell.CurrentConditions(input)
	if err != nil {
		return nil, model.NewAppError("weather plugin", "current-conditions", nil, err.Error(), 500)
	}
	user, _ := p.API.GetUser(args.UserId)
	go p.API.CreatePost(&model.Post{
		Message:   fmt.Sprintf("%s|Requested By|@%s|Query|\"%s\"|", output, user.Username, input),
		UserId:    p.botId,
		ChannelId: args.ChannelId,
		ParentId:  args.ParentId,
	})
	return &model.CommandResponse{}, nil
}
