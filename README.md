# Cine
Utility that let's you convert images and videos (WIP) to ASCII art. (Only tested on Linux right now.)

## Overview
Written in Go, `cine` first scales the input image/gif/video to images/frames of the same size as that of the terminal and then converts those into ASCII and then prints them on the screen in a loop. The output can be displayed in the terminal with color glyphs or as grayscale or as just colors (kinda like `timg` but pixelated), depending on flags included. Oh, it uses ANSI BTW (like how I use arch BTW).

### Features
- Convert images and videos to ASCII art.
- Supports `jpeg`, `jpg`, `png` along with others like `gif` and `video formats in progress.
- Support for colored & grayscale with both color and without color output.
- Customizable character sets for ASCII representation.
- Automatic resizing based on terminal dimensions.

## Installation
Apart from the programming language (Go) itself, the program utilizes the `ffmpeg` for image processing which is called via `os.Exec` so `ffmpeg` should be in `PATH`.

## Usage
To run the program, use the following command format:
```
go run . -file <path_to_image_or_video> [-save <output_directory>] [-grayscale] [-no-font] [-charset <characters_string>]
```
| Option | Description |
| --- | --- |
| `-file` | Path to the image or video file to be converted to ASCII art (required). |
| `-save` | Directory where the "cached" output files will be saved (default: `./cine-out`). |
| `-grayscale` | If specified, the output will be displayed in grayscale instead of color (classic). |
| `-no-font` | If specified, only colors the background without printing any text characters. |
| `-charset` | Custom characters to use for ASCII representation (from lightest to darkest pixels). |

Example: `go run . -file furret.jpg` will print ![furret example](https://github.com/ShubhamVG/cine/GitHub%20Assets/furret.png)

## Known bugs & FAQs
1. **Why are there extra black pixels?** or **The output looks the same as last one even after resizing the terminal. Why?**
Ans. Well, this is a known bug and it is because, the way the program is written, it is supposed to reuse the scaled frames instead of resizing everytime. At the moment, this saving function is not being used but this will be changed soon. A simple way to fix it is to delete the output folder (defaults to `./cine-out`) before running the program or do both (like I do) by using `&&`.

2. Why is there only 1 file?
Ans. I will organize the code some other time.

## TODO
- GIF and video support.
- Reuse the saved folders.
- Add flags like `no-save` or make `save=''` not save anything.
- More flags probably.
- Add more comments and documentation or _hire an unpaid intern to write them for me_ (inside joke).

## Contributions
Fork the repo, make changes, submit PR, you know the drill. If you don't, then learn Git(Hub) or try [my quickstart that I wrote for a friend](https://github.com/ShubhamVG/git-for-maalkin-ji).

## Anything else?
Consider starring and supporting (IDK how) or drop an email or a message on Discord. I may be busy with a hackathon so may not work on this for a while. I hope you could understand my code tho xD.