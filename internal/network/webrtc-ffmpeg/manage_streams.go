package webrtcFfmpeg

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nkien0204/lets-go/internal/configs"
	"github.com/nkien0204/rolling-logger/rolling"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
	"go.uber.org/zap"
)

const (
	h264FrameDuration = time.Millisecond * 33
)

type streamsManager struct {
	mtx         sync.Mutex
	viewers     map[*webrtc.TrackLocalStaticSample]*peerConfig
	isStreaming bool
}

type peerConfig struct {
	uuid           string
	peerConnection *webrtc.PeerConnection
}

var manager *streamsManager

func init() {
	manager = &streamsManager{
		viewers:     make(map[*webrtc.TrackLocalStaticSample]*peerConfig),
		isStreaming: false,
	}
}

func GetManager() *streamsManager {
	return manager
}

func (m *streamsManager) SetupPeer(offer webrtc.SessionDescription) (sdp *webrtc.SessionDescription, rtpSender *webrtc.RTPSender) {
	streamLogger := rolling.New()

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		streamLogger.Error("webrtc.NewPeerConnection failed", zap.Error(err))
		return nil, nil
	}

	// Create a video track
	videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if videoTrackErr != nil {
		streamLogger.Error(videoTrackErr.Error())
		peerConnection.Close()
		return
	}

	uuid := uuid.New().String()
	manager.addStream(videoTrack, peerConnection, uuid)
	logger := streamLogger.With(
		zap.String("uuid", uuid),
	)
	logger.Info("manager: new stream added", zap.Int("number of streams", len(manager.viewers)))

	rtpSender, videoTrackErr = peerConnection.AddTrack(videoTrack)
	if videoTrackErr != nil {
		logger.Error(videoTrackErr.Error())
		return
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		logger.Info("Peer Connection State has changed", zap.String("state", s.String()))

		if s == webrtc.PeerConnectionStateClosed || s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			track, ok := (rtpSender.Track()).(*webrtc.TrackLocalStaticSample)
			if ok {
				manager.removeStream(track)
				logger.Info("a stream removed", zap.Int("number of streams", len(manager.viewers)))
			}
			return
		}
	})

	// Set the remote SessionDescription
	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		logger.Error("peerConnection.SetRemoteDescription failed", zap.Error(err))
		return
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		logger.Error("peerConnection.CreateAnswer failed", zap.Error(err))
		return
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		logger.Error("peerConnection.SetLocalDescription failed", zap.Error(err))
		return
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete
	return peerConnection.LocalDescription(), rtpSender
}

func Streaming(rtpSender *webrtc.RTPSender) {
	logger := rolling.New()

	var dataPipe io.ReadCloser
	var cmd *exec.Cmd
	var err error
	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			rtpSender.SetReadDeadline(time.Now().Add(5 * time.Second))
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				logger.Error("rtpSender.Read failed", zap.Error(rtcpErr))
				if dataPipe != nil {
					dataPipe.Close()
				}
				return
			}
		}
	}()

	go func() {
		args := getFfmpegArgs()
		dataPipe, cmd, err = runCommand("ffmpeg", args...)
		if err != nil {
			logger.Error("RunCommand failed", zap.Error(err))
			return
		}
		defer cmd.Process.Signal(syscall.SIGKILL)

		h264, h264Err := h264reader.NewReader(dataPipe)
		if h264Err != nil {
			logger.Error("h264reader.NewReader failed", zap.Error(h264Err))
			return
		}

		// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
		// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
		//
		// It is important to use a time.Ticker instead of time.Sleep because
		// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
		// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
		spsAndPpsCache := []byte{}
		ticker := time.NewTicker(h264FrameDuration)
		for ; true; <-ticker.C {
			if len(manager.viewers) == 0 {
				manager.SetStream(false)
				dataPipe.Close()
				logger.Info("stop sending video frame")
				return
			}
			nal, h264Err := h264.NextNAL()
			if h264Err == io.EOF {
				logger.Warn("All video frames parsed and sent")
				return
			}
			if h264Err != nil {
				logger.Error("h264.NextNAL failed", zap.Error(err))
				return
			}

			nal.Data = append([]byte{0x00, 0x00, 0x00, 0x01}, nal.Data...)

			if nal.UnitType == h264reader.NalUnitTypeSPS || nal.UnitType == h264reader.NalUnitTypePPS {
				spsAndPpsCache = append(spsAndPpsCache, nal.Data...)
				continue
			} else if nal.UnitType == h264reader.NalUnitTypeCodedSliceIdr {
				nal.Data = append(spsAndPpsCache, nal.Data...)
				spsAndPpsCache = []byte{}
			}

			for videoTrack := range manager.viewers {
				if h264Err = videoTrack.WriteSample(media.Sample{Data: nal.Data, Duration: time.Second}); h264Err != nil {
					logger.Error("videoTrack.WriteSample failed", zap.Error(h264Err))
					return
				}
			}
		}
	}()
}

func getFfmpegArgs() []string {
	webcamConfig := configs.GetConfigs().Webcam
	webcamName := fmt.Sprintf("video=%s", webcamConfig.Name)
	resolution := fmt.Sprintf(webcamConfig.ScreenResolution)
	return []string{"-rtbufsize", "100M", "-f", "dshow", "-i", webcamName, "-pix_fmt", "yuv420p", "-s", resolution, "-c:v", "libx264", "-bsf:v", "h264_mp4toannexb", "-b:v", "2M", "-max_delay", "0", "-bf", "0", "-f", "h264", "-"}
	// return []string{"-f", "dshow", "-i", arg, "-pix_fmt", "yuv420p", "-bf", "0", "-f", "h264", "-"}
}

func runCommand(name string, arg ...string) (io.ReadCloser, *exec.Cmd, error) {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	dataPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	return dataPipe, cmd, nil
}

func (m *streamsManager) addStream(track *webrtc.TrackLocalStaticSample, peer *webrtc.PeerConnection, uuid string) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.viewers[track] = &peerConfig{peerConnection: peer, uuid: uuid}
}

func (m *streamsManager) removeStream(track *webrtc.TrackLocalStaticSample) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if peer, ok := m.viewers[track]; ok {
		peer.peerConnection.Close()
	}
	delete(m.viewers, track)
}

func (m *streamsManager) HasStream() bool {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.isStreaming
}

func (m *streamsManager) SetStream(isStreaming bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.isStreaming = isStreaming
}
