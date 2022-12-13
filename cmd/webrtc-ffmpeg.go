package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nkien0204/lets-go/internal/configs"
	"github.com/nkien0204/lets-go/internal/network/http_handler"
	"github.com/nkien0204/rolling-logger/rolling"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// run WebRTC-FFMPEG server
var runWFCmd = &cobra.Command{
	Use:   "wf",
	Short: ": Run WebRTC-FFMPEG server",
	Run:   runWF,
}

func init() {
	serveCmd.AddCommand(runWFCmd)
}

func runWF(cmd *cobra.Command, args []string) {
	logger := rolling.New()
	defer logger.Sync()

	logger.Info("HTTP server starting...", zap.String("addr", configs.GetConfigs().HttpServer.Address))
	server := http_handler.InitServer()
	go server.ServeHttp()

	// graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	logger.Warn("shutdown app")
}
