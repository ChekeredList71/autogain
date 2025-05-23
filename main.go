package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
)

var errLog = log.New(os.Stdout, "ERR: ", log.Ldate|log.Ltime|log.Lshortfile)
var wg sync.WaitGroup
var command = []string{"custom"}
var rsgainSemaphore chan int

func main() {
	albumMode := flag.Bool("a", false, "Calculate album gain and peak.")
	skipExisting := flag.Bool("S", false, "Don't scan files with existing ReplayGain information.")
	tagMode := flag.String("s", "s", "Tagmode:\ns: scan only\ni: write tags\nd: delete tags")
	targetLoudness := flag.Int("l", -18, "Use n LUFS as target loudness (-30 ≤ n ≤ -5), default: -18")
	clipMode := flag.String("c", "n", "n: no clipping protection (default),\np: clipping protection enabled for positive gain values only,\na: Use max peak level n dB for clipping protection")
	quiet := flag.Bool("q", false, "(rsgain) Don't print scanning status messages.")
	rsgainLimit := flag.Int("r", 100, "Limit, how many rsgain instances can run at a time.")
	flag.Parse()

	libraryRoot := flag.Arg(0)

	// build the rsgain custom command and check values

	if *albumMode {
		command = append(command, "-a")
	}

	if *skipExisting {
		command = append(command, "-S")
	}

	if !slices.Contains([]string{"s", "d", "i"}, *tagMode) {
		fmt.Printf("Invalid clip mode: %s", *tagMode)
		os.Exit(2)
	}
	command = append(command, "-s", *tagMode)

	if !(-30 <= *targetLoudness && *targetLoudness <= -5) {
		fmt.Println("Target loudness n needs to be -30 ≤ n ≤ -5")
		os.Exit(2)
	}
	command = append(command, "-l", strconv.Itoa(*targetLoudness))

	if !slices.Contains([]string{"n", "p", "a"}, *clipMode) {
		fmt.Printf("Invalid clip mode: %s", *clipMode)
		os.Exit(2)
	}
	command = append(command, "-c", *clipMode)

	if libraryRoot == "" {
		fmt.Println("No library path specified.")
		os.Exit(2)
	}

	if *quiet {
		command = append(command, "-q")
	}

	rsgainSemaphore = make(chan int, *rsgainLimit)

	/* Used for debugging
	ctx, cancel := context.WithCancel(context.Background())
	go monitorRsgainProcesses(ctx, 500*time.Millisecond)
	*/

	// scan for album folders
	wg.Add(1)
	err := walker(libraryRoot)

	if err != nil {
		errLog.Printf("Error walking library folder: %s\n", err)
	}

	wg.Wait()

	/*cancel()*/
}

func walker(root string) error {
	defer wg.Done()
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip creating walkers on the initial directory (it would create infinite threads lol)
		if d.IsDir() && path != root {
			// when walked into a directory, launch a new walker on that
			wg.Add(1)
			go func() {
				err := walker(path)
				if err != nil {
					errLog.Printf("Error walking %s: %s\n", path, err)
				}
			}()

			// process supported files (separate thread)
			wg.Add(1)
			go func() {
				defer wg.Done()
				dir, err := os.ReadDir(path)

				if err != nil {
					errLog.Printf("Error reading directory contents: %s\n", err)
					return
				}

				// filter supported audio files
				var audioFiles []string
				for _, file := range dir {
					if isSupportedMusicFile(file.Name()) {
						audioFiles = append(audioFiles, filepath.Join(path, file.Name()))
					}
				}

				if len(audioFiles) > 0 {
					rsgainSemaphore <- 0 //add a slot to the semaphore
					defer func() { <-rsgainSemaphore }()

					cmd := exec.Command("rsgain", append(command, audioFiles...)...)
					err := cmd.Run()

					if err != nil {
						errLog.Printf("Error calling rsgain on these files: '%v'\n", audioFiles)
						errLog.Printf("Command failed: %s\nError: %v\n", cmd.String(), err)

					}
				}
			}()

			// skip the current directory (the newly summoned walker is dealing with it)
			return fs.SkipDir
		}

		return nil
	})
}

func isSupportedMusicFile(path string) bool {
	supportedFiles := []string{
		".aiff", ".flac", ".flac",
		".mp2", ".mp3", ".m4a",
		".mpc", ".ogg", ".oga",
		".spx", ".opus", ".opus",
		".wav", ".wv", ".wma"}

	if slices.Contains(supportedFiles, filepath.Ext(path)) {
		return true
	}

	return false
}

/*
func monitorRsgainProcesses(ctx context.Context, interval time.Duration) {
	var peak int

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cmd := exec.Command("pgrep", "rsgain")
			out, err := cmd.Output()
			if err != nil && len(out) == 0 {
				continue // rsgain not found, 0 instances
			}

			count := bytes.Count(out, []byte("\n"))
			if count > peak {
				peak = count
			}
			log.Printf("Current rsgain instances: %d, Peak: %d\n", count, peak)

		case <-ctx.Done():
			log.Printf("Monitoring stopped. Peak rsgain processes: %d\n", peak)
			return
		}
	}
}
*/
