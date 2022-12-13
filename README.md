# webrtc-ffmpeg
Webcam video streaming with WebRTC and FFMPEG

## How to use
### FFMPEG
- Base on your OS to [download ffmpeg](https://ffmpeg.org/download.html) binary version.
- After downloading, put the `ffmpeg` binary file into the project, (`<PATH>/webrtc-ffmpeg/ffmpeg.exe`) for example.
### Build
```bash
git clone https://github.com/nkien0204/webrtc-ffmpeg.git
go build main.go
```
So now we have these files for running: `main.exe`, `.env`, `ffmpeg.exe` and `index.html`.
Run cmd:
```bash
./main serve wf
```
## Configuration
Take a look into `.env` file
- `LOG_***`: for log config.
- `HTTP_ADDR`: HTTP server address.
- `WEBCAM_NAME`: name of your webcam device.
- `SCREEN_RESOLUTION`: width and height for video (default is 640x360)

**You must write the name of your webcam correctly**

## Testing
Go to browser and enter: `http://<address>/get-player`. It will show a simple player for streaming your webcam!

## Reference
- [Pion-WebRTC](https://github.com/pion/webrtc)
- [ffmpeg-to-webrtc](https://github.com/ashellunts/ffmpeg-to-webrtc)
