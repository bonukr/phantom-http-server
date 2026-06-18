// Command server runs the phantom-http-server HTTP/HTTPS service (API hooks + GUI).
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bonukr/phantom-http-server/internal/config"
	"github.com/bonukr/phantom-http-server/internal/logging"
	"github.com/bonukr/phantom-http-server/internal/server"
	"github.com/bonukr/phantom-http-server/internal/settings"
	"github.com/bonukr/phantom-http-server/web"
)

func main() {
	envCfg := config.Load()

	st, err := settings.Load(envCfg.SettingsFile)
	if err != nil {
		panic(err)
	}

	logger, closer, err := logging.New(st.Log.File, st.Log.Level)
	if err != nil {
		panic(err)
	}
	defer closer.Close()

	logger.Info("settings loaded",
		"file", envCfg.SettingsFile,
		"port", st.Server.Port,
		"tls", st.Server.TLS.Enabled,
		"apis", len(st.APIs),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := server.New(st, logger, web.Assets())

	if err := srv.Run(ctx); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}
