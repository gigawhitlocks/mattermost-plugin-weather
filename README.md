# Notice: This plugin is currently broken

This plugin relies on OpenWeatherMap for base maps and something on their API changed, causing image creation to fail in this plugin. I haven't had the time to investigate it, so for the time being the plugin doesn't work. If you're reading this and would like to use this plugin, I'd love a contribution, but you will probably want to message me and ask for some help because neither this repository nor the weather library repo it draws from are particularly well organized

# A Mattermost Weather Plugin
## Summary
A small Mattermost plugin that shows current weather data using the ClimaCell API for weather data and the OpenCageData API for geocoding.

Insert your ClimaCell and OpenCageData API keys into the System Console. Both companies provide free keys which should be sufficient for small installations.

Here is a brief demonstration of the plugin:

![An example of the plugin in use](./weather-demo.gif)

## Why have a weather plugin? Can't I just visit a weather website? What is the point of this?

You could, but the weather is a topic of conversation that is easy to engage in and relatable for every person. It's the reason that elevator chit-chat can be safely had about the weather to ease uncomfortable silence. It's a great topic to get conversation flowing.

Sometimes in a multi-user chat community it's not obvious if anyone is paying attention in the channel, and you might want to chit-chat. Query the weather bot, and you instantly indicate to your peers that you're there, and also have a low-barrier-to-entry topic to get the conversation rolling, if anyone is around.

## Installation

Download the `.tar.gz` from the [releases page](https://github.com/gigawhitlocks/mattermost-plugin-weather/releases "the releases page is the standard Github location for downloads") and install the plugin through the System Console in Mattermost. The details of that process are beyond the scope of this document.
