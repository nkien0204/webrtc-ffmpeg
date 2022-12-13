package http_handler

import (
	"net/http"

	"github.com/nkien0204/lets-go/internal/configs"
	"github.com/nkien0204/lets-go/internal/network/http_handler/streaming"
	"github.com/nkien0204/rolling-logger/rolling"
	"go.uber.org/zap"
)

type HttpServer struct {
	Address string
}

func InitServer() HttpServer {
	return HttpServer{Address: configs.GetConfigs().HttpServer.Address}
}

func (server *HttpServer) ServeHttp() {
	http.HandleFunc("/streaming", streaming.HandleStreaming)
	http.HandleFunc("/get-player", streaming.HandleGetPlayer)

	if err := http.ListenAndServe(server.Address, nil); err != nil {
		rolling.New().Fatal("ListenAndServe http server failed", zap.Error(err))
	}
}
