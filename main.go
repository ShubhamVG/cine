package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	fontWHRatio = 0.5

	defScaledImgName    = "scaled-out.png"
	colorAsciiChars     = "*"
	grayscaleAsciiChars = "@#%*+=-;:. "
)

func main() {
	defer fmt.Println("\u001b[0m") // reset coloring before exiting

	filePath := flag.String("file", "", "image/video file to asciify")
	saveFolder := flag.String("save", "./cine-out", "folder where files are saved")
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
		fatalExit("Failed to get terminal size")
	}

	// ignore error caused by already exists
	err = os.Mkdir(*saveFolder, 0777)
	if err != nil && !errors.Is(err, os.ErrExist) {
		fatalExit("Failed to create save directory")
	}

	inpFile, err := os.Open(*filePath)
	if err != nil {
		fatalExit("Failed to open -file")
	}

	defer inpFile.Close()

	img, _, err := image.Decode(inpFile)
	if err != nil {
		fatalExit("Failed to read the -file image")
	}

	imgHeight := img.Bounds().Dy()
	imgWidth := img.Bounds().Dx()

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

	var ascii string

	if *useNoFont {
		ascii, err = noFontAsciifyImage(*filePath, *saveFolder, asciiWidth, asciiHeight, *isGrayscale)
	} else {
		ascii, err = asciifyImage(*filePath, *saveFolder, asciiWidth, asciiHeight, *characters, !*isGrayscale)
	}

	if err != nil {
		fatalExit(err.Error())
	}

	//_ = ascii
	fmt.Println(ascii)
}

func asciifyImage(
	filePath string,
	saveFolder string,
	width uint,
	height uint,
	characters string,
	toColor bool,
) (string, error) {
	scaledFilePath := saveFolder + "/" + defScaledImgName
	err := resizeAndSaveImg(filePath, scaledFilePath, width, height)

	if err != nil {
		return "", errors.New("resize image failed")
	}

	imgFile, err := os.Open(scaledFilePath)
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
			r, g, b, _ := pixel.RGBA()

			gray := (r+g+b) / 3
			intensity := float64(gray) / 0xffff
			char := string(characters[int(intensity*float64(len(characters)-1))])

			if toColor {
				r64Str := strconv.FormatUint(uint64(r), 10)
				g64Str := strconv.FormatUint(uint64(g), 10)
				b64Str := strconv.FormatUint(uint64(b), 10)
				rowStr += "\u001b[38;2;" + r64Str + ";" + g64Str + ";" + b64Str + "m" + char
			} else {
				rowStr += char
			}
		}

		asciiString += rowStr + "\n"
	}

	return asciiString, nil
}

func noFontAsciifyImage(
	filePath string,
	saveFolder string,
	width uint,
	height uint,
	isGrayscale bool,
) (string, error) {
	scaledFilePath := saveFolder + "/" + defScaledImgName
	err := resizeAndSaveImg(filePath, scaledFilePath, width, height)

	if err != nil {
		return "", errors.New("resize image failed")
	}

	imgFile, err := os.Open(scaledFilePath)
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
			var r64Str, g64Str, b64Str string
			r, g, b, _ := pixel.RGBA()

			if isGrayscale {
				gray := (r+g+b) / 3
				grayStr := strconv.FormatUint(uint64(gray), 10)
				rowStr += "\u001b[48;2;" + grayStr + ";" + grayStr + ";" + grayStr + "m "
			} else {
				r64Str = strconv.FormatUint(uint64(r), 10)
				g64Str = strconv.FormatUint(uint64(g), 10)
				b64Str = strconv.FormatUint(uint64(b), 10)
				rowStr += "\u001b[48;2;" + r64Str + ";" + g64Str + ";" + b64Str + "m "
			}
		}

		asciiString += rowStr + "\n"
	}

	return asciiString, nil
}

func resizeAndSaveImg(filePath string, savePath string, width uint, height uint) error {
	scaleString := "scale=" + strconv.FormatUint(uint64(width), 10) + ":" + strconv.FormatUint(uint64(height), 10)
	ffmpegCmd := exec.Command("ffmpeg", "-i", filePath, "-vf", scaleString, savePath)
	_, err := ffmpegCmd.Output()

	return err
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
