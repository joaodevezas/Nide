package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/fyne-io/terminal"
)

// ide holds all application state. Using a struct instead of package-level
// globals keeps state contained and avoids the accumulation bugs the old
// version had (e.g. explorer_list growing forever).
type ide struct {
	app    fyne.App
	window fyne.Window

	editor    *widget.Entry
	pathEntry *widget.Entry
	status    *widget.Label

	explorerList *widget.List
	files        []string
	folderPath   string
	currentFile  string

	term          *terminal.Terminal
	termContainer *fyne.Container
	termVisible   bool
	toggleTermBtn *widget.Button

	keybindLabel *widget.Label
}

func newIDE() *ide {
	a := app.New()
	w := a.NewWindow("NIDE")
	w.Resize(fyne.NewSize(1000, 700))

	i := &ide{app: a, window: w, termVisible: true}
	i.build()
	return i
}

func (i *ide) build() {
	i.editor = widget.NewMultiLineEntry()
	i.editor.SetPlaceHolder("Open a folder and select a file to begin editing...")

	i.pathEntry = widget.NewEntry()
	i.pathEntry.SetPlaceHolder("relative/path/to/file.txt")

	i.status = widget.NewLabel("Ready")

	// Clickable file explorer (replaces the old static multi-line Label).
	i.explorerList = widget.NewList(
		func() int { return len(i.files) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(i.files[id])
		},
	)
	i.explorerList.OnSelected = func(id widget.ListItemID) {
		i.openFile(i.files[id])
	}

	// --- Terminal setup ---
	i.term = terminal.New()
	i.termContainer = container.NewMax(i.term)
	go func() {
		if err := i.term.RunLocalShell(); err != nil {
			i.setStatus(fmt.Sprintf("terminal error: %v", err))
			return
		}
		i.setStatus("terminal session ended")
	}()

	openFolderBtn := widget.NewButton("Open Folder", i.chooseFolder)
	saveBtn := widget.NewButton("Save", i.quickSave)
	saveAsBtn := widget.NewButton("Save As", i.saveAs)
	openFileBtn := widget.NewButton("Open Path", i.openFromPathEntry)
	i.toggleTermBtn = widget.NewButton("Hide Terminal", i.toggleTerminal)
	keybindBtn := widget.NewButton("Show Keybinds", i.toggleKeybinds)

	top := container.NewHBox(openFolderBtn, saveBtn, saveAsBtn, openFileBtn, i.toggleTermBtn, keybindBtn)

	i.keybindLabel = widget.NewLabel("")

	// Editor on top, terminal below, resizable via a real split (fixes the
	// old bug where the terminal was crushed to near-zero height inside a VBox).
	editorSplit := container.NewVSplit(i.editor, i.termContainer)
	editorSplit.Offset = 0.6

	pathBar := container.NewBorder(nil, nil, widget.NewLabel("Path:"), nil, i.pathEntry)
	mainArea := container.NewBorder(nil, pathBar, nil, nil, editorSplit)

	explorerScroll := container.NewVScroll(i.explorerList)
	explorerScroll.SetMinSize(fyne.NewSize(220, 0))

	content := container.NewBorder(top, i.status, explorerScroll, i.keybindLabel, mainArea)
	i.window.SetContent(content)

	i.setupShortcuts()
}

// setStatus is safe to call from a goroutine (e.g. the terminal shell
// goroutine). fyne.Do marshals the update onto the UI thread.
// Requires Fyne v2.4+.
func (i *ide) setStatus(s string) {
	fyne.Do(func() {
		i.status.SetText(s)
	})
}

func (i *ide) chooseFolder() {
	d := dialog.NewFolderOpen(func(u fyne.ListableURI, err error) {
		if err != nil || u == nil {
			return
		}
		i.folderPath = u.Path()
		i.refreshExplorer()
	}, i.window)
	d.Show()
}

func (i *ide) refreshExplorer() {
	entries, err := os.ReadDir(i.folderPath)
	if err != nil {
		i.setStatus(err.Error())
		return
	}
	i.files = i.files[:0]
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		i.files = append(i.files, name)
	}
	i.explorerList.Refresh()
}

func (i *ide) openFile(name string) {
	name = strings.TrimSuffix(name, "/")
	full := filepath.Join(i.folderPath, name)

	info, err := os.Stat(full)
	if err != nil {
		i.setStatus(err.Error())
		return
	}
	if info.IsDir() {
		i.folderPath = full
		i.refreshExplorer()
		return
	}

	data, err := os.ReadFile(full)
	if err != nil {
		i.setStatus(err.Error())
		return
	}
	i.editor.SetText(string(data))
	i.currentFile = full
	i.pathEntry.SetText(name)
	i.setStatus("opened " + full)
}

func (i *ide) quickSave() {
	if i.currentFile == "" {
		i.saveAs()
		return
	}
	if err := os.WriteFile(i.currentFile, []byte(i.editor.Text), 0644); err != nil {
		i.setStatus(err.Error())
		return
	}
	i.setStatus("saved " + i.currentFile)
}

func (i *ide) saveAs() {
	if i.pathEntry.Text == "" {
		i.setStatus("enter a path first")
		return
	}
	full := filepath.Join(i.folderPath, i.pathEntry.Text)
	if err := os.WriteFile(full, []byte(i.editor.Text), 0644); err != nil {
		i.setStatus(err.Error())
		return
	}
	i.currentFile = full
	i.refreshExplorer()
	i.setStatus("saved " + full)
}

func (i *ide) openFromPathEntry() {
	if i.pathEntry.Text == "" {
		return
	}
	i.openFile(i.pathEntry.Text)
}

func (i *ide) toggleTerminal() {
	i.termVisible = !i.termVisible
	if i.termVisible {
		i.term.Show()
		i.toggleTermBtn.SetText("Hide Terminal")
	} else {
		i.term.Hide()
		i.toggleTermBtn.SetText("Show Terminal")
	}
}

func (i *ide) toggleKeybinds() {
	if i.keybindLabel.Text == "" {
		i.keybindLabel.SetText(
			"Ctrl+S  Save\n" +
				"Ctrl+O  Save As\n" +
				"Ctrl+R  Open path\n" +
				"Ctrl+F  Open folder\n" +
				"Ctrl+H  Hide/show terminal\n" +
				"Ctrl+L  Toggle this list\n" +
				"Ctrl+Q  Quit",
		)
	} else {
		i.keybindLabel.SetText("")
	}
}

func (i *ide) setupShortcuts() {
	bind := func(key fyne.KeyName, fn func(fyne.Shortcut)) {
		i.window.Canvas().AddShortcut(
			&desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierControl},
			fn,
		)
	}

	bind(fyne.KeyS, func(fyne.Shortcut) { i.quickSave() })
	bind(fyne.KeyO, func(fyne.Shortcut) { i.saveAs() })
	bind(fyne.KeyR, func(fyne.Shortcut) { i.openFromPathEntry() })
	bind(fyne.KeyF, func(fyne.Shortcut) { i.chooseFolder() })
	bind(fyne.KeyH, func(fyne.Shortcut) { i.toggleTerminal() }) // was wrongly bound to L before, colliding with the keybind list
	bind(fyne.KeyL, func(fyne.Shortcut) { i.toggleKeybinds() })
	bind(fyne.KeyQ, func(fyne.Shortcut) { i.app.Quit() })
}

func main() {
	i := newIDE()
	i.window.ShowAndRun() // runs the app loop exactly once — no outer for{}, no double Run()
}
