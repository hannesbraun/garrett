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
	"strings"
)

type garrettTheme struct{}

func (m garrettTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return color.RGBA{
			R: 43,
			G: 176,
			B: 120,
			A: 255,
		}
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
	app := app.New()
	app.Settings().SetTheme(&garrettTheme{})

	w := app.NewWindow("Garrett")

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
	addButton := widget.NewButton("Add...", func() {
		dialog.ShowFileOpen(func(closer fyne.URIReadCloser, err error) {
			if closer == nil || err != nil {
				return
			}

			mimeType, err := mimetype.DetectFile(closer.URI().Path())
			if isSupportedMimeType(mimeType) && err == nil {
				inputFiles.Append(closer.URI().Path())
			} else {
				dialog.ShowInformation("Not supported", "Converting the file "+path.Base(closer.URI().Path())+" is not supported.\nUnsupported type: "+mimeType.String(), w)
			}
		}, w)
	})
	addDirButton := widget.NewButton("Add directory...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil || err != nil {
				return
			}

			items := filesFromDirectory(uri.Path())
			for _, item := range items {
				inputFiles.Append(item)
			}
		}, w)
	})

	removeButton := widget.NewButton("Remove", func() {
		if selected >= inputFiles.Length() {
			return
		}

		in, _ := inputFiles.Get()
		in = append(in[:selected], in[selected+1:]...)
		inputFiles.Set(in)

		if selected >= inputFiles.Length() {
			selected = inputFiles.Length() - 1
		}
	})
	clearButton := widget.NewButton("Clear", func() {
		in, _ := inputFiles.Get()
		inputFiles.Set(in[:0])
		selected = 0
	})
	listControls := container.NewHBox(addButton, addDirButton, removeButton, clearButton)

	outDirLabel := widget.NewLabel("Output directory")
	outDir, err := os.Getwd()
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

	form := container.New(layout.NewFormLayout(), outDirLabel, outDirVal, sampleRateLabel, sampleRateButtons)

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

	content := container.NewBorder(nil, container.NewVBox(listControls, form, widget.NewSeparator(), bottomContainer), nil, nil, list)

	w.SetContent(content)
	w.Resize(fyne.NewSize(768, 700))
	w.ShowAndRun()
}
