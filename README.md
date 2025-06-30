# Screen Recorder (rec)

A simple command-line tool for recording your Windows desktop screen to a video file using [ffmpeg](https://ffmpeg.org/). This tool manages ffmpeg as a subprocess, allowing you to start and stop screen recordings from the terminal.

**Note: This tool is Windows-only.**

## Features
- Start and stop screen recording from the command line
- Records the entire desktop with optional audio (silent by default)
- Customizable frame rate, duration, and output file
- Supports background recording

## Requirements
- **Windows OS** (uses ffmpeg's `gdigrab` input device, which is only available on Windows)
- [ffmpeg](https://ffmpeg.org/) must be installed and available in your system `PATH`
- Go 1.24+ (for building the tool)

## Installation
1. Install [Go](https://golang.org/dl/) (version 1.24 or newer).
2. Install [ffmpeg](https://ffmpeg.org/download.html) and ensure it is in your `PATH`.
3. Clone this repository:
   ```sh
   git clone <repo-url>
   cd video-recording
   ```
4. Build the tool:
   ```sh
   go build -o rec record.go
   ```

## Usage

### Start Recording
```
rec start [options]
```

**Options:**
- `-fps <number>`: Set frames per second (default: 15)
- `-duration <seconds>`: Set maximum duration in seconds (default: unlimited)
- `-output <filename>`: Set output file name (default: `screen.mp4`)
- `-bg`: Run recording in the background

**Examples:**
- Start recording with default settings:
  ```sh
  rec start
  ```
- Record at 30 fps for 60 seconds to `myvideo.mp4`:
  ```sh
  rec start -fps 30 -duration 60 -output myvideo.mp4
  ```
- Start recording in the background:
  ```sh
  rec start -bg
  ```

### Stop Recording
```
rec stop
```
This will gracefully stop the current recording session.

## How It Works
- When you run `rec start`, the tool launches ffmpeg with the appropriate arguments to capture the desktop using `gdigrab`.
- The process ID is saved to a file (`screenrec.pid`).
- To stop recording, `rec stop` creates a kill file (`screenrec.kill`), which signals the running ffmpeg process to stop.

## Limitations
- **Windows only:** The tool uses ffmpeg's `gdigrab` input, which is not available on Linux or macOS. For Linux, use `x11grab` or `kmsgrab` with ffmpeg directly.
- **No audio capture:** The current ffmpeg command uses a silent audio source. You can modify the code to capture system audio if needed.
- **Requires ffmpeg in PATH:** Make sure ffmpeg is installed and accessible from the command line.

## Dependencies
See `go.mod` for Go dependencies. Main external requirement is ffmpeg.

## License
MIT (or specify your license here)

## Credits
- [ffmpeg](https://ffmpeg.org/)
- Inspired by simple screen recording scripts and tools. 