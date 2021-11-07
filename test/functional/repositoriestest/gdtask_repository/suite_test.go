package gdtask_repository

import (
	"testing"

	"github.com/gameap/daemon/internal/app/repositories"
	"github.com/gameap/daemon/test/functional/repositoriestest"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	repositoriestest.Suite

	GDTaskRepository *repositories.GDTasksRepository
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (suite *Suite) SetupSuite() {
	suite.Suite.SetupSuite()

	suite.GDTaskRepository = suite.Container.Get("gdaemonTasksRepository").(*repositories.GDTasksRepository)
}

func (suite *Suite) SetupTest() {
	suite.Suite.SetupTest()
}