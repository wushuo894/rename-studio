package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
)

var droppedFiles = mygo.NewEvent[[]fileItem]("rename-studio:files-dropped")
var dropFailure = mygo.NewEvent[string]("rename-studio:drop-failed")

type Desktop struct {
	session fileSession
}

// Import 打开原生文件选择器，可以一次导入多个文件。
func (d *Desktop) Import(ctx context.Context) ([]fileItem, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: mygo.CallerWindow(ctx), Title: "选择需要重命名的文件", Multiple: true,
	})
	if err != nil {
		return nil, err
	}
	return d.session.importPaths(paths, false)
}

// ConfirmRename 显示原生确认框，按回车或 Esc 时默认取消。
func (d *Desktop) ConfirmRename(ctx context.Context, count int) (bool, error) {
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent: mygo.CallerWindow(ctx), Title: "确认重命名", Type: mygo.MessageWarning,
		Message: fmt.Sprintf("即将重命名 %d 个文件。同名文件会被覆盖，覆盖无法撤销。是否继续？", count),
		Buttons: []string{"重命名", "取消"}, DefaultButton: 1, CancelButton: 1,
	})
	return err == nil && answer.Button == 0, err
}

// ShowMessage 通过调用者的窗口展示原生消息框。
func (d *Desktop) ShowMessage(ctx context.Context, message, kind string) error {
	icon := map[string]mygo.MessageType{
		"info": mygo.MessageInfo, "warning": mygo.MessageWarning, "error": mygo.MessageError,
	}[kind]
	_, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent: mygo.CallerWindow(ctx), Title: "Rename Studio", Message: message, Type: icon, Buttons: []string{"确定"},
	})
	return err
}

// Rename 提交完整批次，由 Go 在同一个会话锁下复核、执行并记录撤销信息。
func (d *Desktop) Rename(items []renameRequest) renameBatch {
	batch, err := d.session.rename(items, false)
	if err != nil {
		// 失败时仍返回磁盘上发生的改动，让文件列表及时更新。
		batch.Error = err.Error()
	}
	return batch
}

// CheckDesktopUpdate 打开中文更新窗口；磁盘操作期间的退出仍由窗口拦截。
func (d *Desktop) CheckDesktopUpdate() { updater.CheckForUpdates() }

// Undo 恢复当前进程最近一次成功的操作。
func (d *Desktop) Undo() error {
	_, err := d.session.rename(nil, true)
	return err
}

// main 在原生主线程创建窗口。扫描拖放内容放到后台，避免阻塞窗口事件。
func main() {
	// 生成绑定时不注册窗口和更新任务，前端资源由 MyGo CLI 统一嵌入。
	if os.Getenv("MYGO_GENERATE") != "" {
		mygo.Bind(&Desktop{})
		if err := mygo.App.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}
	mygo.App.SetName("Rename Studio")
	mygo.Use(updater.New(updater.Options{Language: "zh-Hans"}))
	desktop := &Desktop{session: fileSession{files: make(map[string]importedFile)}}
	mygo.Bind(desktop)

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title: "Rename Studio", URL: "/", Width: 1080, Height: 760,
			MinWidth: 720, MinHeight: 580, BackgroundColor: "#171a1f",
		})
		win.OnFileDrop(func(event *mygo.FileDropEvent) {
			paths := append([]string(nil), event.Paths...)
			go func() {
				// 只有拖放导入会在深度上限内扫描子文件夹。
				items, err := desktop.session.importPaths(paths, true)
				if err != nil {
					_ = dropFailure.Emit(win, err.Error())
					return
				}
				_ = droppedFiles.Emit(win, items)
			}()
		})
		win.OnClose(func(event *mygo.CloseEvent) {
			// 关闭窗口和系统退出时先让正在进行的磁盘操作结束。
			if !desktop.session.mu.TryLock() {
				event.PreventDefault()
				return
			}
			desktop.session.mu.Unlock()
		})
		if runtime.GOOS == "darwin" {
			// macOS 关闭最后一个窗口时结束当前会话，与现有桌面版本一致。
			win.OnClosed(mygo.App.Quit)
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
