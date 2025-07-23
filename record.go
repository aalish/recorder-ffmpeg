package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const (
	pidFileName              = "screenrec.pid"
	killFileName             = "screenrec.kill"
	DETACHED_PROCESS         = 0x00000008
	CREATE_NEW_PROCESS_GROUP = 0x00000200 // Windows process group flag
)

// Add a global variable to hold the path for pid/kill files
var pidKillDir = "."

func pidFilePath() string  { return filepath.Join(pidKillDir, pidFileName) }
func killFilePath() string { return filepath.Join(pidKillDir, killFileName) }

func writePid(pid int) error {
	return ioutil.WriteFile(pidFilePath(), []byte(strconv.Itoa(pid)), 0644)
}

func readPid() (int, error) {
	data, err := ioutil.ReadFile(pidFilePath())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

func removePid() {
	_ = os.Remove(pidFilePath())
}

// startRecording runs ffmpeg, writes its PID, and watches for Ctrl+C or the kill file.
func startRecording(output string, fps int, duration int) error {
	args := []string{"-y"}
	if duration > 0 {
		args = append(args, "-t", strconv.Itoa(duration))
	}
	args = append(args,
		"-f", "gdigrab",
		"-framerate", strconv.Itoa(fps),
		"-i", "desktop",
		"-f", "lavfi", "-i", "anullsrc=channel_layout=stereo:sample_rate=44100",
		"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p",
		"-profile:v", "baseline", "-level", "4.1",
		"-c:a", "aac", "-b:a", "128k",
		"-movflags", "+faststart", "-shortest",
		output,
	)

	cmd := exec.Command("ffmpeg", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// new process group so we can send Ctrl+C
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: CREATE_NEW_PROCESS_GROUP}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	pid := cmd.Process.Pid
	if err := writePid(pid); err != nil {
		return fmt.Errorf("failed to write PID: %w", err)
	}
	fmt.Printf("Recording started (PID=%d).\n", pid)
	fmt.Println("Press Ctrl+C or run `rec stop` to end.")

	// handle Ctrl+C in parent
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		fmt.Println("\nCtrl+C received: sending 'q' to ffmpeg...")
		stdin.Write([]byte("q"))
	}()

	// watch for kill file
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := os.Stat(killFilePath()); err == nil {
				fmt.Println("Kill file detected: sending 'q' to ffmpeg...")
				stdin.Write([]byte("q"))
				_ = os.Remove(killFilePath())
				return
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg exit error: %w", err)
	}
	time.Sleep(2 * time.Second)
	removePid()
	return nil
}

// stopRecording creates the kill file so the recorder will shut down cleanly.
func stopRecording() error {
	if _, err := readPid(); err != nil {
		return fmt.Errorf("could not read PID file: %w", err)
	}
	if err := ioutil.WriteFile(killFilePath(), []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create kill file: %w", err)
	}
	fmt.Println("Kill file created; the recording process will shut down gracefully shortly.")
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: rec start|stop [options]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		// parse flags
		output := "screen.mp4"
		fps := 15
		duration := 0
		bg := false
		path := "."

		// collect args for potential background re-exec
		var reexecArgs []string
		reexecArgs = append(reexecArgs, "start")

		for i := 2; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "-fps":
				i++
				fps, _ = strconv.Atoi(os.Args[i])
				reexecArgs = append(reexecArgs, "-fps", os.Args[i])
			case "-duration":
				i++
				duration, _ = strconv.Atoi(os.Args[i])
				reexecArgs = append(reexecArgs, "-duration", os.Args[i])
			case "-output":
				i++
				output = os.Args[i]
				reexecArgs = append(reexecArgs, "-output", os.Args[i])
			case "-bg":
				bg = true
				// do NOT include -bg in reexecArgs
			case "--path":
				i++
				path = os.Args[i]
				reexecArgs = append(reexecArgs, "--path", os.Args[i])
			default:
				// ignore unknown
			}
		}
		pidKillDir = path

		// if background requested, re-exec detached and exit parent
		if bg {
			cmd := exec.Command(os.Args[0], reexecArgs...)
			cmd.SysProcAttr = &syscall.SysProcAttr{
				CreationFlags: CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS,
			}
			if err := cmd.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to start background process: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Recording started in background (PID=%d). Parent exiting.\n", cmd.Process.Pid)
			os.Exit(0)
		}

		// otherwise run in foreground
		if err := startRecording(output, fps, duration); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "stop":
		path := "."
		for i := 2; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--path":
				i++
				path = os.Args[i]
			}
		}
		pidKillDir = path
		if err := stopRecording(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Println("Unknown command:", os.Args[1])
		fmt.Println("Usage: rec start|stop [options]")
		os.Exit(1)
	}
}

// C:\Users\Administrator\apps\recorder\rec.exe start -bg -fps 15 -output out2.mp4
