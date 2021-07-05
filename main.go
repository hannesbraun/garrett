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
	"image/color"
	"os"
)

type garretTheme struct{}

func (m garretTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
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

func (m garretTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (garretTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (garretTheme) Size(name fyne.ThemeSizeName) float32 {
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
	app.Settings().SetTheme(&garretTheme{})

	w := app.NewWindow("Garret")

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

			inputFiles.Append(closer.URI().Path())
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
	startButton := widget.NewButton("Convert", func() {
		progress.Set(0.0)
		files, _ := inputFiles.Get()
		convert(files, outDir, float64(sampleRate))
		progress.Set(0.99)
	})
	bottomContainer := container.NewBorder(progressBar, nil, nil, startButton, statusLabel)

	content := container.NewBorder(nil, container.NewVBox(listControls, form, widget.NewSeparator(), bottomContainer), nil, nil, list)

	w.SetContent(content)
	w.Resize(fyne.NewSize(768, 700))
	w.ShowAndRun()
}
