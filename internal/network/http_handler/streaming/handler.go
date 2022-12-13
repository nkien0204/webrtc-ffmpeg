package streaming

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nkien0204/rolling-logger/rolling"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

const (
	h264FrameDuration = time.Millisecond * 33
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
	decode(reqBody.Body, &offer)

	remoteSDP, rtpSender := manager.setupPeer(offer)
	if remoteSDP == nil || rtpSender == nil {
		logger.Error("setupPeer failed")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(nil)
		return

	}
	if !manager.hasStream() {
		streaming(rtpSender)
		manager.setStream(true)
	}

	encodedSDP := CamPlayInfo{
		// base64 encode
		Body: encode(remoteSDP),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(encodedSDP)
}
