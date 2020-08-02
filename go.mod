module github.com/gigawhitlocks/mattermost-plugin-weather

go 1.14

replace github.com/gigawhitlocks/weather => ../weather

require (
	github.com/gigawhitlocks/weather v0.0.0-00010101000000-000000000000
	github.com/mattermost/mattermost-server/v5 v5.25.2
	github.com/mholt/archiver/v3 v3.3.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
)
