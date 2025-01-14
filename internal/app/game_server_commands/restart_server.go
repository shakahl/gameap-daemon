package gameservercommands

import (
	"context"
	"io"

	"github.com/gameap/daemon/internal/app/components"
	"github.com/gameap/daemon/internal/app/config"
	"github.com/gameap/daemon/internal/app/contracts"
	"github.com/gameap/daemon/internal/app/domain"
	"github.com/pkg/errors"
)

type defaultRestartServer struct {
	bufCommand
	statusServer contracts.GameServerCommand
	stopServer   contracts.GameServerCommand
	startServer  contracts.GameServerCommand
	baseCommand
}

func newDefaultRestartServer(
	cfg *config.Config,
	executor contracts.Executor,
	processManager contracts.ProcessManager,
	statusServer contracts.GameServerCommand,
	stopServer contracts.GameServerCommand,
	startServer contracts.GameServerCommand,
) *defaultRestartServer {
	cmd := &defaultRestartServer{
		baseCommand:  newBaseCommand(cfg, executor, processManager),
		bufCommand:   bufCommand{output: components.NewSafeBuffer()},
		statusServer: statusServer,
		stopServer:   stopServer,
		startServer:  startServer,
	}

	return cmd
}

func (cmd *defaultRestartServer) Execute(ctx context.Context, server *domain.Server) error {
	cmd.output = components.NewSafeBuffer()

	if cmd.cfg.Scripts.Restart == "" {
		return cmd.restartViaStopStart(ctx, server)
	}

	result, err := cmd.processManager.Restart(ctx, server, cmd.output)
	cmd.SetResult(int(result))
	cmd.SetComplete()

	return err
}

func (cmd *defaultRestartServer) restartViaStopStart(ctx context.Context, server *domain.Server) error {
	defer cmd.SetComplete()

	err := cmd.statusServer.Execute(ctx, server)
	if err != nil {
		return errors.WithMessage(err, "failed to check server status")
	}
	active := cmd.statusServer.Result() == SuccessResult

	if active {
		err = cmd.stopServer.Execute(ctx, server)
		if err != nil {
			return errors.WithMessage(err, "failed to stop server")
		}

		if cmd.stopServer.Result() != SuccessResult {
			cmd.SetResult(cmd.stopServer.Result())
			return nil
		}
	}

	err = cmd.startServer.Execute(ctx, server)
	if err != nil {
		return err
	}

	cmd.SetResult(cmd.startServer.Result())

	return nil
}

func (cmd *defaultRestartServer) ReadOutput() []byte {
	var err error
	var out []byte

	if cmd.cfg.Scripts.Restart == "" {
		statusOut := cmd.statusServer.ReadOutput()
		stopOut := cmd.stopServer.ReadOutput()
		startOut := cmd.startServer.ReadOutput()
		out = append(out, statusOut...)
		out = append(out, stopOut...)
		out = append(out, startOut...)
	} else {
		out, err = io.ReadAll(cmd.output)
		if err != nil {
			return []byte(err.Error())
		}
	}

	return out
}
