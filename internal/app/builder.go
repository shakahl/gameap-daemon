package app

import (
	"context"
	"time"

	"github.com/gameap/daemon/internal/app/components"
	"github.com/gameap/daemon/internal/app/config"
	"github.com/gameap/daemon/internal/app/domain"
	gameservercommands "github.com/gameap/daemon/internal/app/game_server_commands"
	gdscheduler "github.com/gameap/daemon/internal/app/gdaemon_scheduler"
	"github.com/gameap/daemon/internal/app/interfaces"
	"github.com/gameap/daemon/internal/app/repositories"
	"github.com/go-resty/resty/v2"
	"github.com/sarulabs/di"
	log "github.com/sirupsen/logrus"
)

func NewBuilder(cfg *config.Config, logger *log.Logger) (*di.Builder, error) {
	builder, err := di.NewBuilder()
	if err != nil {
		return nil, err
	}

	err = builder.Add(definitions(cfg, logger)...)
	if err != nil {
		return nil, err
	}

	return builder, nil
}

const (
	restyDef        = "resty"
	cacheManagerDef = "cacheManager"
	storeDef        = "store"
	apiCallerDef    = "apiCaller"
	executorDef     = "executorDef"

	gdaemonTaskRepositoryDef = "gdaemonTasksRepository"
	serverRepositoryDef      = "serverRepository"
	serverTaskRepositoryDef  = "serverTaskRepository"

	serverCommandFactoryDef = "serverCommandFactory"

	gdTaskMangerDef = "gdTaskManager"
)

//nolint:funlen
func definitions(cfg *config.Config, logger *log.Logger) []di.Def {
	return []di.Def{
		{
			Name: cacheManagerDef,
			Build: func(ctn di.Container) (interface{}, error) {
				return NewLocalCache(cfg)
			},
		},
		{
			Name: storeDef,
			Build: func(ctn di.Container) (interface{}, error) {
				return NewLocalStore(cfg)
			},
		},
		{
			Name: apiCallerDef,
			Build: func(ctn di.Container) (interface{}, error) {
				return NewAPICaller(
					context.TODO(),
					cfg,
					ctn.Get(restyDef).(*resty.Client),
				)
			},
		},
		{
			Name: restyDef,
			Build: func(ctn di.Container) (interface{}, error) {
				restyClient := resty.New()
				restyClient.SetHostURL(cfg.APIHost)
				restyClient.SetHeader("User-Agent", "GameAP Daemon/3.0")
				restyClient.RetryCount = 30
				restyClient.RetryMaxWaitTime = 10 * time.Minute
				restyClient.SetLogger(logger)

				return restyClient, nil
			},
		},
		{
			Name: executorDef,
			Build: func(ctn di.Container) (interface{}, error) {
				return components.NewExecutor(), nil
			},
		},
		// Repositories
		{
			Name: gdaemonTaskRepositoryDef,
			Build: func(ctn di.Container) (interface{}, error) {
				apiClient := ctn.Get(apiCallerDef).(interfaces.APIRequestMaker)
				serverRepository := ctn.Get(serverRepositoryDef).(domain.ServerRepository)

				return repositories.NewGDTaskRepository(
					apiClient,
					serverRepository,
				), nil
			},
		},
		{
			Name: serverRepositoryDef,
			Build: func(ctn di.Container) (interface{}, error) {
				apiClient := ctn.Get(apiCallerDef).(interfaces.APIRequestMaker)

				return repositories.NewServerRepository(apiClient), nil
			},
		},
		{
			Name: serverTaskRepositoryDef,
			Build: func(ctn di.Container) (interface{}, error) {
				apiClient := ctn.Get(apiCallerDef).(interfaces.APIRequestMaker)
				serverRepository := ctn.Get(serverRepositoryDef).(domain.ServerRepository)

				return repositories.NewServerTaskRepository(apiClient, serverRepository), nil
			},
		},
		// Factories
		{
			Name: serverCommandFactoryDef,
			Build: func(ctn di.Container) (interface{}, error) {
				serverRepository := ctn.Get(serverRepositoryDef).(domain.ServerRepository)
				executor := ctn.Get(executorDef).(interfaces.Executor)

				return gameservercommands.NewFactory(
					cfg,
					serverRepository,
					executor,
				), nil
			},
		},
		// Services
		{
			Name: gdTaskMangerDef,
			Build: func(ctn di.Container) (interface{}, error) {
				return gdscheduler.NewTaskManager(
					ctn.Get(gdaemonTaskRepositoryDef).(domain.GDTaskRepository),
					ctn.Get(cacheManagerDef).(interfaces.Cache),
					ctn.Get(serverCommandFactoryDef).(*gameservercommands.ServerCommandFactory),
					cfg,
				), nil
			},
		},
	}
}
