package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"
	"github.com/nfnt/resize"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Steam Screenshots Importer")

	steamGameIDEntry := widget.NewEntry()
	steamGameIDEntry.SetPlaceHolder("Enter Steam Game ID (e.g., 11111111)")
	steamGameIDEntry.Validator = func(text string) error {
		if len(text) == 0 {
			return fmt.Errorf("steam game ID cannot be empty")
		}
		if !isNumeric(text) {
			return fmt.Errorf("steam game ID must be numeric")
		}
		return nil
	}

	steamUserdataPathEntry := widget.NewEntry()
	steamUserdataPathEntry.SetPlaceHolder("Enter Steam userdata folder path (e.g., D:\\Steam\\userdata\\1111111)")
	steamUserdataPathEntry.Validator = func(text string) error {
		if !isValidPath(text) {
			return fmt.Errorf("invalid steam userdata path")
		}
		return nil
	}

	originScreenshotsPathEntry := widget.NewEntry()
	originScreenshotsPathEntry.SetPlaceHolder("Enter Origin screenshots folder path (e.g., D:\\screenshots\\some_game)")
	originScreenshotsPathEntry.Validator = func(text string) error {
		if !isValidPath(text) {
			return fmt.Errorf("invalid origin screenshots path")
		}
		return nil
	}

	openFolderDialog := func(entry *widget.Entry) func() {
		return func() {
			dir, err := zenity.SelectFile(zenity.Title("Select Folder"), zenity.Directory())
			if err == nil && dir != "" {
				entry.SetText(dir)
			}
		}
	}

	steamUserdataBrowseButton := widget.NewButton("Browse", openFolderDialog(steamUserdataPathEntry))
	originScreenshotsBrowseButton := widget.NewButton("Browse", openFolderDialog(originScreenshotsPathEntry))

	importButton := widget.NewButton("Import Screenshots", func() {
		steamGameID := steamGameIDEntry.Text
		steamUserdataPath := steamUserdataPathEntry.Text
		originScreenshotsPath := originScreenshotsPathEntry.Text

		if steamGameID == "" || steamUserdataPath == "" || originScreenshotsPath == "" {
			dialog.ShowError(fmt.Errorf("all fields must be filled"), myWindow)
			return
		}

		steamScreenshotsFolderPath := filepath.Join(steamUserdataPath, "760", "remote", steamGameID, "screenshots")
		thumbnailsPath := filepath.Join(steamScreenshotsFolderPath, "thumbnails")

		if err := os.MkdirAll(thumbnailsPath, os.ModePerm); err != nil {
			dialog.ShowError(err, myWindow)
			return
		}

		err := processScreenshots(originScreenshotsPath, steamScreenshotsFolderPath)
		if err != nil {
			dialog.ShowError(err, myWindow)
			return
		}

		dialog.ShowInformation("Success", "Screenshots imported to Steam folder. Please restart Steam to apply changes!", myWindow)
	})

	updateImportButtonState := func() {
		if isValidSteamGameID(steamGameIDEntry.Text) && isValidPath(steamUserdataPathEntry.Text) && isValidPath(originScreenshotsPathEntry.Text) {
			importButton.Enable()
		} else {
			importButton.Disable()
		}
	}

	steamGameIDEntry.OnChanged = func(s string) {
		updateImportButtonState()
	}

	steamUserdataPathEntry.OnChanged = func(s string) {
		updateImportButtonState()
	}

	originScreenshotsPathEntry.OnChanged = func(s string) {
		updateImportButtonState()
	}

	importButton.Disable()

	form := container.NewVBox(
		widget.NewLabel("Steam Game ID:"),
		steamGameIDEntry,
		widget.NewLabel("Steam Userdata Folder Path:"),
		steamUserdataPathEntry,
		steamUserdataBrowseButton,
		widget.NewLabel("Origin Screenshots Folder Path:"),
		originScreenshotsPathEntry,
		originScreenshotsBrowseButton,
		importButton,
	)

	myWindow.SetContent(form)
	myWindow.Resize(fyne.NewSize(600, 400))
	myWindow.ShowAndRun()
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func isValidPath(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func isValidSteamGameID(steamGameID string) bool {
	return len(steamGameID) > 0 && isNumeric(steamGameID)
}

func processScreenshots(originPath, steamPath string) error {
	files, err := os.ReadDir(originPath)
	if err != nil {
		return err
	}

	var screenshots []os.DirEntry
	for _, file := range files {
		if strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") ||
			strings.HasSuffix(strings.ToLower(file.Name()), ".jpeg") ||
			strings.HasSuffix(strings.ToLower(file.Name()), ".png") {
			screenshots = append(screenshots, file)
		}
	}

	sort.Slice(screenshots, func(i, j int) bool {
		iInfo, _ := screenshots[i].Info()
		jInfo, _ := screenshots[j].Info()
		return iInfo.ModTime().Before(jInfo.ModTime())
	})

	nameMap := make(map[string]int)
	for _, file := range screenshots {
		filePath := filepath.Join(originPath, file.Name())
		info, _ := file.Info()

		baseName := info.ModTime().Format("20060102150405")
		nameMap[baseName]++
		finalName := fmt.Sprintf("%s_%d.jpg", baseName, nameMap[baseName])

		destPath := filepath.Join(steamPath, finalName)

		if strings.HasSuffix(strings.ToLower(file.Name()), ".png") {
			if err := convertPngToJpg(filePath, destPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(filePath, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func convertPngToJpg(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	img, _, err := image.Decode(srcFile)
	if err != nil {
		return err
	}

	outFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer outFile.Close()

	resizedImg := resize.Resize(0, 0, img, resize.Lanczos3)
	return jpeg.Encode(outFile, resizedImg, &jpeg.Options{Quality: 100})
}

func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	img, _, err := image.Decode(srcFile)
	if err != nil {
		return err
	}

	outFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return jpeg.Encode(outFile, img, &jpeg.Options{Quality: 100})
}
