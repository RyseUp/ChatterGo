package app

import (
	"github.com/RyseUp/ChatterGo/config"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type App struct {
	Engine *gin.Engine
	DB     *gorm.DB
	Cfg    *config.Config
}

func New(engine *gin.Engine, db *gorm.DB, cfg *config.Config) *App {
	_ = db.AutoMigrate()

	return &App{
		Engine: engine,
		DB:     db,
		Cfg:    cfg,
	}
}

func (a *App) RegisterRoutes() {
	a.Engine.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(
			200, gin.H{"ok": true},
		)
	})
}
