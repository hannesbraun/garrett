package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gabriel-vasile/mimetype"
	"image/color"
	"os"
	"path"
	"strconv"
	"strings"
)

// Custom theming

var primaryColor = color.RGBA{
	R: 0,
	G: 0,
	B: 0,
	A: 0,
}

type garrettTheme struct{}

func (m garrettTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary && primaryColor.A != 0 {
		return primaryColor
	}

	return theme.DefaultTheme().Color(name, variant)
}

func (m garrettTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (garrettTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (garrettTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 12
	case theme.SizeNameInlineIcon:
		return 16
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func main() {
	if len(os.Args) > 1 {
		value, err := strconv.ParseInt(strings.TrimPrefix(os.Args[1], "0x"), 16, 64)
		if err == nil {
			primaryColor = color.RGBA{
				R: uint8((0xFF0000 & value) >> 16),
				G: uint8((0x00FF00 & value) >> 8),
				B: uint8(0x0000FF & value),
				A: 255,
			}
		}
	}

	garrettApp := app.New()
	garrettApp.Settings().SetTheme(&garrettTheme{})

	w := garrettApp.NewWindow("Garrett")

	// File list
	inputFiles := binding.BindStringList(
		&[]string{},
	)
	selected := 0
	list := widget.NewListWithData(inputFiles,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		selected = id
	}
	list.OnUnselected = func(id widget.ListItemID) {
		selected = inputFiles.Length() - 1
	}

	// List control buttons
	addButton := widget.NewButton("Add...", func() {
		dialog.ShowFileOpen(func(closer fyne.URIReadCloser, err error) {
			if closer == nil || err != nil {
				return
			}

			mimeType, err := mimetype.DetectFile(closer.URI().Path())
			if isSupportedMimeType(mimeType) && err == nil {
				err := inputFiles.Append(closer.URI().Path())
				if err != nil {
					dialog.ShowInformation("Error", "Unable to append the file "+path.Base(closer.URI().Path())+" to the list", w)
				}
			} else {
				mimeTypeStr := "<unknown>"
				if mimeType != nil {
					mimeTypeStr = mimeType.String()
				}
				dialog.ShowInformation("Not supported", "Converting the file "+path.Base(closer.URI().Path())+" is not supported.\nUnsupported type: "+mimeTypeStr, w)
			}
		}, w)
	})
	addDirButton := widget.NewButton("Add directory...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil || err != nil {
				return
			}

			items := filesFromDirectory(uri.Path())
			errOccurred := false
			for _, item := range items {
				err := inputFiles.Append(item)
				if err != nil {
					errOccurred = true
				}
			}

			if errOccurred {
				dialog.ShowInformation("Error", "Unable to append some files", w)
			}
		}, w)
	})
	removeButton := widget.NewButton("Remove", func() {
		if selected >= inputFiles.Length() {
			return
		}

		in, _ := inputFiles.Get()
		in = append(in[:selected], in[selected+1:]...)
		err := inputFiles.Set(in)
		if err != nil {
			dialog.ShowInformation("Error", "Unable to remove the file "+path.Base(in[selected]), w)
			return
		}

		if selected >= inputFiles.Length() {
			selected = inputFiles.Length() - 1
		}
	})
	clearButton := widget.NewButton("Clear", func() {
		in, _ := inputFiles.Get()
		err := inputFiles.Set(in[:0])
		if err != nil {
			dialog.ShowInformation("Error", "Unable to remove the file "+path.Base(in[selected]), w)
			return
		}
		selected = 0
	})
	listControls := container.NewHBox(addButton, addDirButton, removeButton, clearButton)

	// Output
	outDirLabel := widget.NewLabel("Output directory")
	outDir, err := os.UserHomeDir()
	if err != nil {
		outDir = "/"
	}
	outDirLabel2 := widget.NewLabel(outDir)
	selectOutDirButton := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil || err != nil {
				return
			}

			outDir = uri.Path()
			outDirLabel2.SetText(outDir)
		}, w)
	})
	outDirVal := container.NewBorder(nil, nil, nil, selectOutDirButton, outDirLabel2)

	// Sample rate selection
	sampleRateLabel := widget.NewLabel("Sample rate")
	sampleRate := 48000
	sampleRateButtons := widget.NewRadioGroup([]string{"44100 Hz", "48000 Hz"}, func(val string) {
		if val == "44100 Hz" {
			sampleRate = 44100
		} else {
			sampleRate = 48000
		}
	})
	sampleRateButtons.SetSelected("48000 Hz")

	// Assemble form
	form := container.New(layout.NewFormLayout(), outDirLabel, outDirVal, sampleRateLabel, sampleRateButtons)

	// Status and control "bar"
	progressVal := 0.0
	progress := binding.BindFloat(&progressVal)
	progressBar := widget.NewProgressBarWithData(progress)
	statusLabel := widget.NewLabel("Idle")
	running := false
	startButton := widget.NewButton("Convert", func() {
		if running {
			return
		}
		running = true

		files, _ := inputFiles.Get()
		go func(files []string, outDir string, sampleRate int, progress *binding.ExternalFloat, statusLabel **widget.Label) {
			failed := convert(files, outDir, float64(sampleRate), progress, statusLabel)
			if len(failed) > 0 {
				filesStr := strings.Join(failed, "\n")
				dialog.ShowInformation("Unable to convert the following files", filesStr, w)
			}
			running = false
		}(files, outDir, sampleRate, &progress, &statusLabel)
	})
	bottomContainer := container.NewBorder(progressBar, nil, nil, startButton, statusLabel)

	// Put everything together and run it
	content := container.NewBorder(nil, container.NewVBox(listControls, form, widget.NewSeparator(), bottomContainer), nil, nil, list)
	w.SetContent(content)
	w.Resize(fyne.NewSize(768, 700))
	w.ShowAndRun()
}
