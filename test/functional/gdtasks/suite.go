package gdtasks

import (
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/gameap/daemon/internal/app/components"
	"github.com/gameap/daemon/internal/app/config"
	"github.com/gameap/daemon/internal/app/contracts"
	"github.com/gameap/daemon/internal/app/domain"
	gameservercommands "github.com/gameap/daemon/internal/app/game_server_commands"
	gdaemonscheduler "github.com/gameap/daemon/internal/app/gdaemon_scheduler"
	"github.com/gameap/daemon/internal/app/services"
	"github.com/gameap/daemon/test/functional"
	"github.com/gameap/daemon/test/mocks"
)

type Suite struct {
	functional.GameServerSuite

	TaskManager      *gdaemonscheduler.TaskManager
	GDTaskRepository *mocks.GDTaskRepository
	ServerRepository *mocks.ServerRepository
	Executor         contracts.Executor
	Cache            contracts.Cache
	Cfg              *config.Config

	WorkPath string
}

func (suite *Suite) SetupSuite() {
	var err error

	suite.GDTaskRepository = mocks.NewGDTaskRepository()
	suite.ServerRepository = mocks.NewServerRepository()

	suite.Executor = components.NewExecutor()

	suite.Cfg = &config.Config{
		Scripts: config.Scripts{
			Start: "{command}",
			Stop:  "{command}",
		},
	}

	suite.Cache, err = services.NewLocalCache(suite.Cfg)
	if err != nil {
		suite.T().Fatal(err)
	}

	suite.TaskManager = gdaemonscheduler.NewTaskManager(
		suite.GDTaskRepository,
		suite.Cache,
		gameservercommands.NewFactory(
			suite.Cfg,
			suite.ServerRepository,
			suite.Executor,
		),
		components.NewCleanExecutor(),
		suite.Cfg,
	)
}

func (suite *Suite) RunTaskManager(timeout time.Duration) {
	suite.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := suite.TaskManager.Run(ctx)
	if err != nil {
		suite.T().Fatal(err)
	}
}

func (suite *Suite) AssertGDTaskExist(task *domain.GDTask) {
	suite.T().Helper()

	actualTask, err := suite.GDTaskRepository.FindByID(context.Background(), task.ID())
	if err != nil {
		suite.T().Fatal(err)
	}

	suite.Require().NotNil(actualTask)
	suite.Assert().Equal(task.Status(), actualTask.Status())
	suite.Assert().Equal(task.RunAfterID(), actualTask.RunAfterID())
	suite.Assert().Equal(task.Task(), actualTask.Task())
	if task.Server() != nil {
		suite.Assert().Equal(task.Server().ID(), actualTask.Server().ID())
	} else {
		suite.Assert().Nil(actualTask.Server())
	}
}

func (suite *Suite) GivenGDTaskWithCommand(cmd string) *domain.GDTask {
	minID := 100
	maxID := 1000000000
	task := domain.NewGDTask(
		rand.Intn(maxID-minID)+minID,
		0,
		nil,
		domain.GDTaskCommandExecute,
		cmd,
		domain.GDTaskStatusWaiting,
	)

	suite.GDTaskRepository.Set([]*domain.GDTask{task})

	return task
}

func (suite *Suite) GivenGDTaskWithIDForServer(id int, server *domain.Server) *domain.GDTask {
	task := domain.NewGDTask(
		id,
		0,
		server,
		domain.GDTaskGameServerStart,
		"",
		domain.GDTaskStatusWaiting,
	)

	suite.GDTaskRepository.Set([]*domain.GDTask{task})

	return task
}

func (suite *Suite) GivenSequenceGDTaskForServer(server *domain.Server) []*domain.GDTask {
	suite.T().Helper()

	tasks := []*domain.GDTask{
		domain.NewGDTask(
			2,
			1,
			server,
			domain.GDTaskGameServerStop,
			"",
			domain.GDTaskStatusWaiting,
		),
		domain.NewGDTask(
			7,
			5,
			server,
			domain.GDTaskGameServerStart,
			"",
			domain.GDTaskStatusWaiting,
		),
		domain.NewGDTask(
			3,
			2,
			server,
			domain.GDTaskGameServerStop,
			"",
			domain.GDTaskStatusWaiting,
		),
		domain.NewGDTask(
			1,
			0,
			server,
			domain.GDTaskGameServerStart,
			"",
			domain.GDTaskStatusWaiting,
		),
		domain.NewGDTask(
			5,
			3,
			server,
			domain.GDTaskGameServerStart,
			"",
			domain.GDTaskStatusWaiting,
		),
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(tasks), func(i, j int) { tasks[i], tasks[j] = tasks[j], tasks[i] })

	suite.GDTaskRepository.Set(tasks)

	return tasks
}

func (suite *Suite) AssertFileContents(file string, expectedContents []byte) {
	suite.T().Helper()

	suite.Require().FileExists(file)

	fd, err := os.Open(file)
	if err != nil {
		suite.T().Fatal(err)
	}
	defer func() {
		if err = fd.Close(); err != nil {
			suite.T().Fatal(err)
		}
	}()

	buf, err := ioutil.ReadAll(fd)

	suite.Assert().Equal(expectedContents, buf)
}
