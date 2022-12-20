package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var Terminal_info string
var path string = "nothing"

var keybind_list string

var content *fyne.Container

var file_path string
var folder_path string

var explorer_list string
var left *widget.Label = widget.NewLabel("")
var terminal_hide bool = false
var left_scroll *container.Scroll

//var left_box *fyne.Container

func main() {
	for {
		//Terminal_info = "Nothing"

		app := app.New()
		//app := headless.NewApp()
		window := app.NewWindow("NIDE")
		window.Resize(fyne.NewSize(800, 960))

		text_input := widget.NewMultiLineEntry()

		//console
		console := widget.NewEntry()

		//terminal

		terminal := widget.NewMultiLineEntry()
		terminal_run := &desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: fyne.KeyModifierControl}
		window.Canvas().AddShortcut(terminal_run, func(shortcut fyne.Shortcut) {

			if runtime.GOOS == "linux" {
				shell, err := exec.Command(terminal.Text).Output()
				if err != nil {
					fmt.Println(err)
				}
				terminal.SetText("\n" + string(shell) + "\n")
			}

			if runtime.GOOS == "windows" {
				powershell, err := exec.Command("powershell", terminal.Text).Output()
				if err != nil {
					fmt.Println(err)
				}
				terminal.SetText(string(powershell))

			}

		})

		terminal.SetText("terminal")

		bottom := container.NewVBox()
		bottom.Add(console)
		bottom.Add(terminal)

		terminal_button := widget.NewButton("Hide terminal", func() {
			terminal_hide = !terminal_hide

			if terminal_hide {
				terminal.Hide()
			} else {
				terminal.Show()
			}
		})

		folder_button := widget.NewButton("Open Folder", func() {
			folderDialog := dialog.NewFileOpen(func(folder fyne.URIReadCloser, err error) {
				if err == nil && folder != nil {
					file_path = folder.URI().String()
					folder_path = string(file_path)
					folder_lenght := len(folder_path)

					for string(folder_path[folder_lenght-1]) != "/" {
						folder_path = folder_path[:folder_lenght-1]
						folder_lenght = len(folder_path)
					}

					folder_path = strings.Replace(folder_path, "file://", "", 1)

					files, err := ioutil.ReadDir(folder_path)
					if err != nil {
						panic(err)
					}

					for _, f := range files {
						explorer_list += f.Name()
						explorer_list += "\n"
					}

					left.SetText(explorer_list)
					left.Refresh()

				}
			}, window)

			folderDialog.Show()

		})
		//keybind button

		right := widget.NewLabel("")
		keybind_list := false

		top := container.NewHBox()

		keybind_button := widget.NewButton("Show Keybinds", func() {
			keybind_list = !keybind_list

			if keybind_list {
				right.SetText("^S| Quick Save \n ^O| Write Out \n ^R| Read file \n ^Z| Exit \n ^E| Run terminal")
			} else {
				right.SetText("")
			}
		})
		top.Add(keybind_button)
		top.Add(folder_button)
		top.Add(terminal_button)

		content = container.NewBorder(top, bottom, left, right, text_input)

		//quicksave keybind
		ctrlSave := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
		window.Canvas().AddShortcut(ctrlSave, func(shortcut fyne.Shortcut) {
			Terminal_info = "Text saved"
			console.SetPlaceHolder(Terminal_info)
			os.WriteFile(path, []byte(text_input.Text), 0666)
		})

		//save keybind
		ctrlOffer := &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierControl}
		window.Canvas().AddShortcut(ctrlOffer, func(shortcut fyne.Shortcut) {
			Terminal_info = "Write to file: "
			console.SetPlaceHolder(Terminal_info)
			path = console.Text
			path = folder_path + path
			os.WriteFile(path, []byte(text_input.Text), 0666)
		})

		//read file
		Open_file := &desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierControl}
		window.Canvas().AddShortcut(Open_file, func(shortcut fyne.Shortcut) {
			console.SetPlaceHolder(Terminal_info)
			path = console.Text
			path = folder_path + path
			read_file, err := os.ReadFile(path)
			if err != nil {
				panic(err)
			}
			text_input.Text = string(read_file)
			text_input.Refresh()
		})

		//Quit the application
		Quit := &desktop.CustomShortcut{KeyName: fyne.KeyZ, Modifier: fyne.KeyModifierControl}
		window.Canvas().AddShortcut(Quit, func(shortcut fyne.Shortcut) {
			app.Quit()
		})

		window.SetContent(content)
		window.ShowAndRun()
		app.Run()
	}

}
