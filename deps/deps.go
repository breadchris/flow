package deps

import (
	"github.com/breadchris/flow/config"
	"github.com/breadchris/flow/session"
	"gorm.io/gorm"
)

type Deps struct {
	Dir     string
	DB      *gorm.DB
	Config  config.AppConfig
	Session *session.SessionManager
}