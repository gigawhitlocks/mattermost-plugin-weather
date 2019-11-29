package main

import (
	"testing"

	"github.com/gigawhitlocks/weather/nws"
	"github.com/stretchr/testify/assert"
)

func TestWeather(t *testing.T) {
	assert := assert.New(t)
	r, err := nws.GetWeather("78703")
	assert.NoError(err)
	assert.NotNil(r)
	assert.NotEqual("", r.String())
}
