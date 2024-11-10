package main

import (
	"errors"
	"flag"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AsciiConverter func(
	infilePath string,
	saveFolder string,
	width uint,
	height uint,
	characters string,
	toColor bool,
) (string, error)

const (
	fontWHRatio = 0.5

	colorAsciiChars     = "*"
	grayscaleAsciiChars = "@#%*+=-;:. "
)

func main() {
	start := time.Now()
	defer fmt.Println("\u001b[0m") // reset coloring before exiting

	infilePath := flag.String("file", "", "image/video file to asciify")
	isVerbose := flag.Bool("verbose", false, "print verbose logs")
	saveFolder := flag.String("save", "", "folder where files are saved")
	isGrayscale := flag.Bool("grayscale", false, "display as grayscaled or not")
	useNoFont := flag.Bool("no-font", false, "only color the background with no text")
	characters := flag.String("charset", "", "characters to use (for lightest to darkest pixels)")
	flag.Parse()

	if *characters == "" {
		if *isGrayscale {
			*characters = grayscaleAsciiChars
		} else {
			*characters = colorAsciiChars
		}
	}

	termWidth, termHeight, err := getTerminalSize()
	if err != nil {
		if *isVerbose {
			fmt.Println(err)
		}
		fatalExit("Failed to get terminal size")
	} else {
		// let's not take the entire space of the terminal
		// TODO: add a flag to adjust this
		termHeight = uint(float64(termHeight) * 0.75)
	}

	fileExt := filepath.Ext(*infilePath)

	if *saveFolder == "" {
		*saveFolder = "cine-out"
		extStartingIndex := len(*infilePath) - len(fileExt)
		fileNameNoExt := (*infilePath)[:extStartingIndex]
		fileNameNoExt += strconv.FormatUint(uint64(termWidth), 10)
		fileNameNoExt += "x" + strconv.FormatUint(uint64(termHeight), 10)
		*saveFolder += "-" + fileNameNoExt
	}

	// ignore error caused by already exists
	err = os.Mkdir(*saveFolder, 0770)
	if errors.Is(err, os.ErrExist) {
		// TODO: fetch the frames and skip rendering

	} else if err != nil {
		if *isVerbose {
			fmt.Println(err)
		}
		fatalExit("Failed to create save directory")
	}

	imgWidth, imgHeight, err := getVMediaFileDimensions(*infilePath)
	if err != nil {
		fatalExit("Failed to get input file's dimensions")
	}

	termWHRatio := float64(termWidth) / float64(termHeight)
	imgWHRatio := float64(imgWidth) / float64(imgHeight)

	asciiHeight := uint(0)
	asciiWidth := uint(0)

	if imgWHRatio > termWHRatio {
		asciiWidth = termWidth
		asciiHeight = uint(float64(asciiWidth) * (1 / imgWHRatio) * fontWHRatio)
	} else {
		asciiHeight = termHeight
		asciiWidth = uint(float64(asciiHeight) * imgWHRatio * (1 / fontWHRatio))
	}

	fmt.Println(time.Since(start))
	start = time.Now()

	numberOfFrames, err := resizeAndSaveFrames(*infilePath, *saveFolder, asciiWidth, asciiHeight)
	if err != nil {
		if *isVerbose {
			fmt.Println(err)
		}
		fatalExit("Failed to extract frames from -file")
	}

	asciiFrames := make([]string, numberOfFrames)
	var wg sync.WaitGroup
	wg.Add(int(numberOfFrames))

	fmt.Println(time.Since(start))
	start = time.Now()

	if *useNoFont {
		for frameNumber := range numberOfFrames {
			go func(i uint, wgPtr *sync.WaitGroup) {
				defer wgPtr.Done()

				fName := *saveFolder + "/frame-" + fmt.Sprintf("%06d", i+1) + ".png"
				ascii, err := noFontAsciifyImage(fName, asciiWidth, asciiHeight, *isGrayscale)

				if err != nil {
					if *isVerbose {
						fmt.Println(err)
					}
					fatalExit(err.Error())
				}

				asciiFrames[i] = ascii
			}(frameNumber, &wg)
		}
	} else {
		for frameNumber := range numberOfFrames {
			go func(i uint, wgPtr *sync.WaitGroup) {
				defer wgPtr.Done()

				fName := *saveFolder + "/frame-" + fmt.Sprintf("%06d", i+1) + ".png"
				ascii, err := asciifyImage(fName, asciiWidth, asciiHeight, *characters, !*isGrayscale)

				if err != nil {
					if *isVerbose {
						fmt.Println(err)
					}
					fatalExit(err.Error())
				}

				asciiFrames[i] = ascii
			}(frameNumber, &wg)
		}
	}

	wg.Wait()
	fmt.Println(time.Since(start))

	// go to linestart and then no %d lines up
	restartCommand := fmt.Sprintf("\r\u001b[%dA", termHeight)

	if numberOfFrames > 1 {
		for {
			for frameNumber := range numberOfFrames {
				fmt.Print(asciiFrames[frameNumber])
				time.Sleep(time.Millisecond * 50) // TODO: make dynamic FPS
				fmt.Print(restartCommand)
			}
		}
	} else {
		fmt.Println(asciiFrames[0])
	}
}

func asciifyImage(
	infilePath string,
	width uint,
	height uint,
	characters string,
	toColor bool,
) (string, error) {
	imgFile, err := os.Open(infilePath)
	if err != nil {
		return "", errors.New("ffmpeg-scaled image opening failed")
	}

	defer imgFile.Close()

	img, err := png.Decode(imgFile)
	if err != nil {
		return "", err
	}

	asciiString := ""
	for y := range height {
		rowStr := ""

		for x := range width {
			pixel := img.At(int(x), int(y))
			r32, g32, b32, _ := pixel.RGBA()
			r := uint8(r32 / 256)
			g := uint8(g32 / 256)
			b := uint8(b32 / 256)

			gray := (r + g + b) / 3
			intensity := float64(gray) / 0xff
			char := string(characters[int(intensity*float64(len(characters)-1))])

			if toColor {
				rAsStr := strconv.FormatUint(uint64(r), 10)
				gAsStr := strconv.FormatUint(uint64(g), 10)
				bAsStr := strconv.FormatUint(uint64(b), 10)
				rowStr += "\u001b[38;2;" + rAsStr + ";" + gAsStr + ";" + bAsStr + "m" + char
			} else {
				rowStr += char
			}
		}

		asciiString += rowStr + "\n"
	}

	return asciiString, nil
}

func noFontAsciifyImage(
	infilePath string,
	width uint,
	height uint,
	isGrayscale bool,
) (string, error) {
	imgFile, err := os.Open(infilePath)
	if err != nil {
		return "", errors.New("ffmpeg-scaled image opening failed")
	}

	defer imgFile.Close()

	img, err := png.Decode(imgFile)
	if err != nil {
		return "", err
	}

	asciiString := ""
	for y := range height {
		rowStr := ""

		for x := range width {
			pixel := img.At(int(x), int(y))
			r32, g32, b32, _ := pixel.RGBA()
			r := uint8(r32 / 256)
			g := uint8(g32 / 256)
			b := uint8(b32 / 256)

			if isGrayscale {
				gray := (r + g + b) / 3
				grayStr := strconv.FormatUint(uint64(gray), 10)
				rowStr += "\u001b[48;2;" + grayStr + ";" + grayStr + ";" + grayStr + "m "
			} else {
				rAsStr := strconv.FormatUint(uint64(r), 10)
				gAsStr := strconv.FormatUint(uint64(g), 10)
				bAsStr := strconv.FormatUint(uint64(b), 10)
				rowStr += "\u001b[48;2;" + rAsStr + ";" + gAsStr + ";" + bAsStr + "m "
			}
		}

		asciiString += rowStr + "\n"
	}

	return asciiString, nil
}

func resizeAndSaveFrames(infilePath string, saveFolder string, width uint, height uint) (uint, error) {
	scaleString := "scale=" + strconv.FormatUint(uint64(width), 10) + ":" + strconv.FormatUint(uint64(height), 10)
	ffmpegCmd := exec.Command("ffmpeg", "-i", infilePath, "-vf", scaleString, saveFolder+"/frame-%06d.png")
	_, err := ffmpegCmd.Output()

	if err != nil {
		return 0, errors.New("failed to extract & resize frame")
	}

	dir, err := os.Open(saveFolder)
	if !errors.Is(err, os.ErrExist) && err != nil {
		return 0, err
	}

	files, err := dir.ReadDir(-1) // -1 means read everything

	return uint(len(files)), err
}

func fatalExit(msg string) {
	fmt.Println("Fatal: " + msg)
	os.Exit(1)
}

func getTerminalSize() (uint, uint, error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()

	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(string(out))

	if len(parts) != 2 {
		return 0, 0, errors.New("unexpected stty size returned values")
	}

	colSize, colErr := strconv.ParseUint(parts[0], 10, 32)
	rowSize, rowErr := strconv.ParseUint(parts[1], 10, 32)

	if rowErr != nil || colErr != nil {
		return 0, 0, errors.New("row/column parsing uint failed")
	}

	return uint(rowSize), uint(colSize), nil
}

func getVMediaFileDimensions(infilePath string) (uint, uint, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "stream=width,height", "-of", "csv=p=0:s=' '", infilePath)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()

	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(string(out))
	if len(parts) != 2 {
		return 0, 0, errors.New("expected result from ffprobe while getting dimensions: " + string(out))
	}

	width, widthErr := strconv.ParseUint(parts[0], 10, 32)
	height, heightErr := strconv.ParseUint(parts[1], 10, 32)

	if widthErr != nil || heightErr != nil {
		return 0, 0, errors.New("vmedia width/height parsing uint failed")
	}

	return uint(width), uint(height), nil
}
