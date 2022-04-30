package orgmanager

import (
	"github.com/larksuite/oapi-sdk-go/core"
	"github.com/larksuite/oapi-sdk-go/core/config"
)

type Feishu struct {
	oapiConfig *config.Config
	config     *feishuConfig
}

type feishuConfig struct {
	AppID     string
	AppSecret string
}

func (f Feishu) Init() {
	appSettings := core.NewInternalAppSettings(
		core.SetAppCredentials(f.config.AppID, f.config.AppSecret),
	)
	f.oapiConfig = core.NewConfig(core.DomainFeiShu, appSettings, core.SetLoggerLevel(core.LoggerLevelError))
}
