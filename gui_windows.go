package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/lxn/walk"
	d "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

type audioFileModel struct {
	walk.TableModelBase
	paths []string
}

func (model *audioFileModel) RowCount() int {
	return len(model.paths)
}

func (model *audioFileModel) Value(row, column int) interface{} {
	switch column {
	case 0:
		return row + 1
	case 1:
		return filepath.Base(model.paths[row])
	default:
		return filepath.Dir(model.paths[row])
	}
}

type audioWindow struct {
	window       *walk.MainWindow
	controls     *walk.Composite
	files        *audioFileModel
	table        *walk.TableView
	bitrate      *walk.ComboBox
	output       *walk.LineEdit
	status       *walk.Label
	removeButton *walk.PushButton
	clearButton  *walk.PushButton
	upButton     *walk.PushButton
	downButton   *walk.PushButton
	mergeButton  *walk.PushButton
	busy         bool
	cancel       context.CancelFunc
	done         chan struct{}
	uiMutex      sync.Mutex
	closing      bool
}

func runGUI() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	app := &audioWindow{files: &audioFileModel{}}
	err := (d.MainWindow{
		AssignTo: &app.window,
		Title:    "Simple Audio Merger",
		MinSize:  d.Size{Width: 700, Height: 480},
		Size:     d.Size{Width: 860, Height: 560},
		Font:     d.Font{Family: "Segoe UI", PointSize: 10},
		Layout:   d.VBox{Margins: d.Margins{Left: 16, Top: 16, Right: 16, Bottom: 16}, Spacing: 12},
		Children: []d.Widget{
			d.Label{Text: "Объединение аудио", Font: d.Font{Family: "Segoe UI", PointSize: 16, Bold: true}},
			d.Label{Text: "Добавьте MP3 и WAV, расположите их в нужном порядке и сохраните один MP3"},
			d.Composite{
				AssignTo: &app.controls,
				Layout:   d.VBox{MarginsZero: true, Spacing: 12},
				Children: []d.Widget{
					d.Composite{
						Layout: d.HBox{MarginsZero: true},
						Children: []d.Widget{
							d.PushButton{Text: "Добавить файлы…", OnClicked: app.addFiles},
							d.PushButton{AssignTo: &app.removeButton, Text: "Удалить", OnClicked: app.removeFile},
							d.PushButton{AssignTo: &app.clearButton, Text: "Очистить", OnClicked: app.clearFiles},
							d.HSpacer{},
						},
					},
					d.Composite{
						Layout:        d.HBox{MarginsZero: true},
						StretchFactor: 1,
						Children: []d.Widget{
							d.TableView{
								AssignTo:                 &app.table,
								Model:                    app.files,
								AlternatingRowBG:         true,
								LastColumnStretched:      true,
								NotSortableByHeaderClick: true,
								OnCurrentIndexChanged:    app.updateActions,
								MinSize:                  d.Size{Width: 400, Height: 180},
								Columns: []d.TableViewColumn{
									{Title: "№", Width: 40},
									{Title: "Файл", Width: 250},
									{Title: "Папка", Width: 300},
								},
							},
							d.Composite{
								Layout: d.VBox{MarginsZero: true},
								Children: []d.Widget{
									d.PushButton{AssignTo: &app.upButton, Text: "Вверх", OnClicked: func() { app.moveFile(-1) }},
									d.PushButton{AssignTo: &app.downButton, Text: "Вниз", OnClicked: func() { app.moveFile(1) }},
									d.VSpacer{},
								},
							},
						},
					},
					d.Composite{
						Layout: d.HBox{MarginsZero: true},
						Children: []d.Widget{
							d.Label{Text: "Битрейт MP3:"},
							d.ComboBox{
								AssignTo:     &app.bitrate,
								Model:        []string{"32", "40", "48", "56", "64", "80", "96", "112", "128", "160", "192", "224", "256", "320"},
								CurrentIndex: 13,
								MinSize:      d.Size{Width: 90},
							},
							d.Label{Text: "кбит/с"},
							d.HSpacer{},
						},
					},
					d.Composite{
						Layout: d.HBox{MarginsZero: true},
						Children: []d.Widget{
							d.Label{Text: "Результат:"},
							d.LineEdit{AssignTo: &app.output, ReadOnly: true, CueBanner: "Выберите, куда сохранить MP3"},
							d.PushButton{Text: "Выбрать…", OnClicked: func() { app.chooseOutput() }},
						},
					},
				},
			},
			d.Composite{
				Layout: d.HBox{MarginsZero: true},
				Children: []d.Widget{
					d.Label{AssignTo: &app.status, Text: "Добавьте аудиофайлы"},
					d.HSpacer{},
					d.PushButton{AssignTo: &app.mergeButton, Text: "Объединить", MinSize: d.Size{Width: 140}, OnClicked: app.merge},
				},
			},
		},
	}).Create()
	if err != nil {
		walk.MsgBox(nil, "Simple Audio Merger", "Не удалось открыть окно:\n"+err.Error(), walk.MsgBoxIconError)
		return err
	}
	defer app.window.Dispose()
	app.updateActions()
	app.window.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		app.uiMutex.Lock()
		app.closing = true
		app.uiMutex.Unlock()
		if app.cancel != nil {
			app.cancel()
		}
	})
	app.window.Run()
	if app.cancel != nil {
		app.cancel()
	}
	if app.done != nil {
		<-app.done
	}
	return nil
}

func (app *audioWindow) addFiles() {
	dialog := walk.FileDialog{
		Title:  "Выберите аудиофайлы",
		Filter: "Аудиофайлы (*.mp3;*.wav)|*.mp3;*.wav|MP3 (*.mp3)|*.mp3|WAV (*.wav)|*.wav",
	}
	accepted, err := dialog.ShowOpenMultiple(app.window)
	if err != nil {
		app.showError(err)
		return
	}
	if !accepted {
		return
	}
	for _, path := range dialog.FilePaths {
		if !isSupportedAudio(path) {
			app.showError(fmt.Errorf("поддерживаются только MP3 и WAV: %s", path))
			return
		}
	}
	first := len(app.files.paths)
	app.files.paths = append(app.files.paths, dialog.FilePaths...)
	app.refreshFiles(first)
}

func (app *audioWindow) removeFile() {
	index := app.table.CurrentIndex()
	if index < 0 || index >= len(app.files.paths) {
		return
	}
	app.files.paths = append(app.files.paths[:index], app.files.paths[index+1:]...)
	app.refreshFiles(index)
}

func (app *audioWindow) clearFiles() {
	app.files.paths = nil
	app.refreshFiles(-1)
}

func (app *audioWindow) moveFile(direction int) {
	index := app.table.CurrentIndex()
	target := index + direction
	if index < 0 || target < 0 || target >= len(app.files.paths) {
		return
	}
	app.files.paths[index], app.files.paths[target] = app.files.paths[target], app.files.paths[index]
	app.refreshFiles(target)
}

func (app *audioWindow) refreshFiles(index int) {
	app.files.PublishRowsReset()
	if index >= len(app.files.paths) {
		index = len(app.files.paths) - 1
	}
	app.table.SetCurrentIndex(index)
	app.updateActions()
	app.status.SetText(fmt.Sprintf("Файлов: %d", len(app.files.paths)))
}

func (app *audioWindow) updateActions() {
	if app.mergeButton == nil {
		return
	}
	count := len(app.files.paths)
	index := app.table.CurrentIndex()
	selected := index >= 0 && index < count && !app.busy
	app.removeButton.SetEnabled(selected)
	app.clearButton.SetEnabled(count > 0 && !app.busy)
	app.upButton.SetEnabled(selected && index > 0)
	app.downButton.SetEnabled(selected && index < count-1)
	app.mergeButton.SetEnabled(count > 0 && !app.busy)
}

func (app *audioWindow) chooseOutput() bool {
	path := app.output.Text()
	if path == "" {
		path = "merged_audio.mp3"
		if len(app.files.paths) > 0 {
			path = filepath.Join(filepath.Dir(app.files.paths[0]), path)
		}
	}
	dialog := walk.FileDialog{
		Title:    "Сохранить объединённый MP3",
		FilePath: path,
		Filter:   "MP3 (*.mp3)|*.mp3",
		Flags:    win.OFN_PATHMUSTEXIST,
	}
	accepted, err := dialog.ShowSave(app.window)
	if err != nil {
		app.showError(err)
		return false
	}
	if !accepted {
		return false
	}
	path = dialog.FilePath
	if filepath.Ext(path) == "" {
		path += ".mp3"
	}
	if !strings.EqualFold(filepath.Ext(path), ".mp3") {
		app.showError(fmt.Errorf("файл результата должен иметь расширение .mp3"))
		return false
	}
	app.output.SetText(path)
	return true
}

func (app *audioWindow) merge() {
	if app.busy || len(app.files.paths) == 0 {
		return
	}
	if app.output.Text() == "" && !app.chooseOutput() {
		return
	}
	output := app.output.Text()
	overwrite := false
	if _, err := os.Stat(output); err == nil {
		answer := walk.MsgBox(app.window, "Заменить файл?", "Файл уже существует:\n"+output+"\n\nЗаменить его?", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion|walk.MsgBoxDefButton2)
		if answer != win.IDYES {
			return
		}
		overwrite = true
	} else if !os.IsNotExist(err) {
		app.showError(err)
		return
	}
	bitrate, err := strconv.Atoi(app.bitrate.Text())
	if err != nil {
		app.showError(fmt.Errorf("выберите битрейт MP3"))
		return
	}
	paths := append([]string(nil), app.files.paths...)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	app.cancel, app.done = cancel, done
	app.busy = true
	app.controls.SetEnabled(false)
	app.updateActions()
	app.status.SetText("Объединение файлов…")
	go func() {
		defer close(done)
		defer cancel()
		err := mergeFiles(ctx, paths, output, bitrate, overwrite)
		app.synchronize(func() {
			if app.window.IsDisposed() {
				return
			}
			app.busy = false
			app.cancel = nil
			app.controls.SetEnabled(true)
			app.updateActions()
			if err != nil {
				app.status.SetText("Не удалось объединить файлы")
				app.showError(err)
				return
			}
			app.status.SetText("Готово — файл сохранён")
			walk.MsgBox(app.window, "Готово", "Аудиофайлы объединены:\n"+output, walk.MsgBoxIconInformation)
		})
	}()
}

func (app *audioWindow) synchronize(callback func()) {
	app.uiMutex.Lock()
	defer app.uiMutex.Unlock()
	if !app.closing {
		app.window.Synchronize(callback)
	}
}

func (app *audioWindow) showError(err error) {
	walk.MsgBox(app.window, "Ошибка", err.Error(), walk.MsgBoxIconError)
}
