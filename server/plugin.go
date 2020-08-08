package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/gigawhitlocks/weather/climacell"
	"github.com/google/uuid"
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
	err = p.API.RegisterCommand(&model.Command{
		Trigger:          "weather",
		DisplayName:      "Weather",
		Description:      "Gets the weather",
		AutoComplete:     true,
		AutoCompleteDesc: "Search for a location like \"1600 Pennsylvania Ave\", \"Miami\", \"Purdue University\", or even \"The Statue of Liberty\".",
	})
	if err != nil {
		return err
	}
	err = p.API.RegisterCommand(&model.Command{
		Trigger:          "weathermap",
		DisplayName:      "Weathermap",
		Description:      "Gets a weather map for a location",
		AutoComplete:     true,
		AutoCompleteDesc: "Search for a location like \"Brooklyn\" and include any of the following -precipitation, -temp, -wind_speed, -wind_direction, -wind_gust, -visibility, -baro_pressure , -dewpoint, -humidity, -cloud_cover, -cloud_base, -cloud_ceiling, -cloud_satellite e.g. /weather brooklyn -precipitation",
	})
	return err
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	w.Write(p.profileImage)
}

var featureFlagPattern *regexp.Regexp = regexp.MustCompile(`-[a-zA-Z_]+`)

func (p *Plugin) prepareMapInput(input string) (string, []string) {
	input = strings.TrimPrefix(input, "map ")
	arguments := []string{}
	for match := featureFlagPattern.FindStringIndex(input); match != nil; match = featureFlagPattern.FindStringIndex(input) {
		argument := strings.TrimPrefix(input[match[0]:match[1]], "-")
		arguments = append(arguments, argument)
		input = input[:match[0]] + input[match[1]:]
	}
	return input, arguments
}

func (p *Plugin) getMap(c *plugin.Context, args *model.CommandArgs, input string) (*model.CommandResponse, *model.AppError) {
	// weathermap

	user, _ := p.API.GetUser(args.UserId)
	post := &model.Post{
		Message:   fmt.Sprintf("|Requested By|Query|\n|----|-----|\n|@%s|\"%s\"|", user.Username, strings.TrimPrefix(input, "map ")),
		UserId:    p.botId,
		ChannelId: args.ChannelId,
		ParentId:  args.ParentId,
	}

	location, features := p.prepareMapInput(input)
	precipMap, err := climacell.BuildMap(location, features...)
	if err != nil {
		return nil, model.NewAppError("getMap", err.Error(), nil, err.Error(), 500)
	}
	fileInfo, ferr := p.API.UploadFile(
		precipMap, args.ChannelId,
		fmt.Sprintf("%s.png", uuid.New().String()))

	if ferr == nil {
		post.FileIds = []string{fileInfo.Id}
	}

	p.API.CreatePost(post)
	return &model.CommandResponse{}, nil
}

func (p *Plugin) getCurrentConditions(c *plugin.Context, args *model.CommandArgs, input string) (*model.CommandResponse, *model.AppError) {
	output, err := climacell.CurrentConditions(input)
	if err != nil {
		return nil, model.NewAppError("weather plugin", err.Error(), nil, err.Error(), 500)
	}
	user, _ := p.API.GetUser(args.UserId)
	post := &model.Post{
		Message:   fmt.Sprintf("%s|Requested By|@%s|Query|\"%s\"|", output, user.Username, input),
		UserId:    p.botId,
		ChannelId: args.ChannelId,
		ParentId:  args.ParentId,
	}

	p.API.CreatePost(post)
	return &model.CommandResponse{}, nil
}
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	input := strings.TrimSpace(strings.TrimPrefix(args.Command, "/weather"))
	if strings.HasPrefix(input, "map") {
		return p.getMap(c, args, input)
	}
	input = strings.TrimSpace(input)
	return p.getCurrentConditions(c, args, input)

}
