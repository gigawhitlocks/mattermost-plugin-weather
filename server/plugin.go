package main

import (
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"strings"
	"sync"

	"github.com/gigawhitlocks/weather/nws"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

func (p *Plugin) OnActivate() error {
	// args.Command contains the full command string entered
	return p.API.RegisterCommand(&model.Command{
		Trigger:          "weather",
		DisplayName:      "Weather",
		Description:      "Gets the weather from the National Weather Service (US only) by zip code",
		AutoComplete:     true,
		AutoCompleteDesc: "/weather 78703 would return the weather for downtown Austin, TX.",
	})
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	zip := strings.TrimSpace(strings.TrimPrefix(args.Command, "/weather "))
	if len(zip) != 5 {
		return nil, model.NewAppError("weather plugin", "zip", nil, "input wasn't length 5", 400)
	}

	currentConditions, err := nws.GetWeather(zip)
	if err != nil {
		return nil, model.NewAppError("weather plugin", "getweather", nil, err.Error(), 401)
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		Text:         currentConditions.String(),
	}, nil
}
