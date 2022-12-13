package configs

import (
	"errors"
	"os"
	"sync"

	"github.com/nkien0204/rolling-logger/rolling"
	"go.uber.org/zap"
)

type Cfg struct {
	HttpServer HttpServerConfig
	Webcam     WebcamConfig
}

type HttpServerConfig struct {
	Address string
}

type WebcamConfig struct {
	Name             string
	ScreenResolution string
}

var config *Cfg
var once sync.Once

// Singleton pattern
func GetConfigs() *Cfg {
	once.Do(func() {
		var err error
		config = initConfigs()
		if err = validateConfig(config); err != nil {
			rolling.New().Error("initConfigs failed", zap.Error(err))
			panic(1)
		}
	})
	return config
}

func initConfigs() *Cfg {
	return &Cfg{
		HttpServer: loadHttpServerConfig(),
		Webcam:     loadWebcamConfig(),
	}
}

func validateConfig(cfg *Cfg) error {
	if cfg.HttpServer.Address == "" {
		return errors.New("could not load http server config")
	}
	if cfg.Webcam.Name == "" {
		return errors.New("could not load webcam config")
	}
	if cfg.Webcam.ScreenResolution == "" {
		cfg.Webcam.ScreenResolution = "640x360" //default
	}
	return nil
}

func loadWebcamConfig() WebcamConfig {
	return WebcamConfig{
		Name:             os.Getenv("WEBCAM_NAME"),
		ScreenResolution: os.Getenv("SCREEN_RESOLUTION"),
	}
}

func loadHttpServerConfig() HttpServerConfig {
	return HttpServerConfig{
		Address: os.Getenv("HTTP_ADDR"),
	}
}
