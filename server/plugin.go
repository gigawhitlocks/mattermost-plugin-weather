package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
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

// ServeHTTP handles HTTP requests to the plugin.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch path := r.URL.Path; path {
	case "/profile.png":
		p.handleProfileImage(w, r)
	default:
		http.NotFound(w, r)
	}
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
	obsv, err := climacell.CurrentConditions(input)
	if err != nil {
		return nil, model.NewAppError("weather plugin", err.Error(), nil, err.Error(), 500)
	}

	attachments := []*model.SlackAttachment{
		{
			Id:        0,
			Title:     fmt.Sprintf("Weather For %s", obsv.ParsedLocation),
			TitleLink: "",
			Fields: []*model.SlackAttachmentField{
				{
					Title: "Conditions",
					Value: fmt.Sprintf("%s", obsv.Title()),
					Short: false,
				},
				{
					Title: "Temperature",
					Value: fmt.Sprintf("%.1f° %s", obsv.Temp.Value, obsv.Temp.Units),
					Short: true,
				},
				{
					Title: "Feels Like",
					Value: fmt.Sprintf("%.1f° %s", obsv.FeelsLike.Value, obsv.FeelsLike.Units),
					Short: true,
				},
				{
					Title: "Type of Precipitation",
					Value: fmt.Sprintf("%s", obsv.PrecipitationType.Value),
					Short: true,
				},
				{
					Title: "Amount of Precipitation",
					Value: fmt.Sprintf("%.1f", obsv.Precipitation.Value),
					Short: true,
				},
				{
					Title: "Wind Gust",
					Value: fmt.Sprintf("%.1f %s", obsv.WindGust.Value, obsv.WindGust.Units),
					Short: true,
				},
				{
					Title: "Barometric Pressure",
					Value: fmt.Sprintf("%.1f %s", obsv.BaroPressure.Value, obsv.BaroPressure.Units),
					Short: true,
				},
				{
					Title: "Humidity",
					Value: fmt.Sprintf("%.1f %s", obsv.Humidity.Value, obsv.Humidity.Units),
					Short: true,
				},
				{
					Title: "Cloud Cover",
					Value: fmt.Sprintf("%.1f %s", obsv.CloudCover.Value, obsv.CloudCover.Units),
					Short: true,
				},
			},
		},
	}

	if attachments[0].Fields[4].Value == "0.0" { // brittle but whatever
		attachments[0].Fields = append(attachments[0].Fields[:3], attachments[0].Fields[5:]...)
	}

	return &model.CommandResponse{
		ResponseType:   model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Username:       "weather",
		ChannelId:      args.ChannelId,
		IconURL:        fmt.Sprintf("/plugins/%s/profile.png", manifest.ID),
		Attachments:    attachments,
		ExtraResponses: nil,
	}, nil
}
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	input := strings.TrimSpace(strings.TrimPrefix(args.Command, "/weather"))
	if strings.HasPrefix(input, "map") {
		return p.getMap(c, args, input)
	}
	input = strings.TrimSpace(input)
	return p.getCurrentConditions(c, args, input)

}

func (p *Plugin) handleProfileImage(w http.ResponseWriter, r *http.Request) {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		p.API.LogError("Unable to get bundle path, err=" + err.Error())
		return
	}

	img, err := os.Open(filepath.Join(bundlePath, "assets", "weather.png"))
	if err != nil {
		http.NotFound(w, r)
		p.API.LogError("Unable to read profile image, err=" + err.Error())
		return
	}
	defer img.Close()

	w.Header().Set("Content-Type", "image/png")
	io.Copy(w, img)
}
