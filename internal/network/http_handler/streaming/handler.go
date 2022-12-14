package streaming

import (
	"encoding/json"
	"net/http"

	webrtcFfmpeg "github.com/nkien0204/lets-go/internal/network/webrtc-ffmpeg"
	"github.com/nkien0204/rolling-logger/rolling"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type CamPlayInfo struct {
	Body string `json:"encodedSDP"`
}

func HandleGetPlayer(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func HandleStreaming(w http.ResponseWriter, r *http.Request) {
	logger := rolling.New()
	var reqBody CamPlayInfo
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		logger.Error("decode request failed", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// base64 decode
	offer := webrtc.SessionDescription{}
	webrtcFfmpeg.Decode(reqBody.Body, &offer)

	manager := webrtcFfmpeg.GetManager()
	remoteSDP, rtpSender := manager.SetupPeer(offer)
	if remoteSDP == nil || rtpSender == nil {
		logger.Error("setupPeer failed")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(nil)
		return

	}

	// Only one connection to the webcam
	if !manager.HasStream() {
		webrtcFfmpeg.Streaming(rtpSender)
		manager.SetStream(true)
	}

	encodedSDP := CamPlayInfo{
		// base64 encode
		Body: webrtcFfmpeg.Encode(remoteSDP),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(encodedSDP)
}
