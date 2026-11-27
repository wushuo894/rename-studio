package main

import (
	"os"
	"path/filepath"
	"testing"
)

// 用不同的内容区分文件身份，并使用真实路径避免临时目录本身的系统链接干扰断言。
func renameFixture(t *testing.T) (*fileSession, string, []fileItem) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"A.txt", "B.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	session := &fileSession{files: make(map[string]importedFile)}
	items, err := session.importPaths([]string{filepath.Join(dir, "A.txt"), filepath.Join(dir, "B.txt")}, false)
	if err != nil || len(items) != 2 {
		t.Fatalf("导入失败：%v, %v", items, err)
	}
	return session, dir, items
}

// 比较文件内容，确认改名及撤销移动的是原来的文件。
func assertFile(t *testing.T, dir, name, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil || string(got) != want {
		t.Fatalf("%s 内容为 %q，应为 %q（%v）", name, got, want, err)
	}
}

// 先腾出被另一项占用的目标，撤销时按反向依赖顺序恢复。
func TestRenameChainAndUndo(t *testing.T) {
	session, dir, items := renameFixture(t)
	batch, err := session.rename([]renameRequest{{items[0].ID, "B.txt"}, {items[1].ID, "C.txt"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	results := batch.Changes
	if len(results) != 2 || results[0].Source != items[0].SourcePath || results[1].Source != items[1].SourcePath {
		t.Fatalf("返回结果必须保留前端提交顺序：%v", results)
	}
	assertFile(t, dir, "B.txt", "A.txt")
	assertFile(t, dir, "C.txt", "B.txt")
	if _, err := session.rename(nil, true); err != nil {
		t.Fatal(err)
	}
	assertFile(t, dir, "A.txt", "A.txt")
	assertFile(t, dir, "B.txt", "B.txt")
}

// 没有空闲名称时拒绝整个互换批次，两个原文件都保留。
func TestRenameRejectsSwap(t *testing.T) {
	session, dir, items := renameFixture(t)
	if _, err := session.rename([]renameRequest{{items[0].ID, "B.txt"}, {items[1].ID, "A.txt"}}, false); err == nil {
		t.Fatal("应拒绝无法直接完成的名称互换")
	}
	assertFile(t, dir, "A.txt", "A.txt")
	assertFile(t, dir, "B.txt", "B.txt")
}

// 模拟执行前复核结束后出现的外部文件，要求拒绝覆盖并回滚此前成功的步骤。
func TestLateTargetConflictRollsBack(t *testing.T) {
	session, dir, items := renameFixture(t)
	dirs := make(map[string]*os.File)
	steps, err := session.prepare([]renameRequest{{items[0].ID, "C.txt"}, {items[1].ID, "D.txt"}}, dirs, false)
	for _, handle := range dirs {
		defer handle.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "D.txt"), []byte("外部文件"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runSteps(steps, false); err == nil {
		t.Fatal("目标已经存在时应失败")
	}
	assertFile(t, dir, "D.txt", "外部文件")
	assertFile(t, dir, "A.txt", "A.txt")
	assertFile(t, dir, "B.txt", "B.txt")
	if _, err := os.Lstat(filepath.Join(dir, "C.txt")); !os.IsNotExist(err) {
		t.Fatalf("第一个最终名称未回滚：%v", err)
	}
}

// 已导入的同名目标允许被覆盖，覆盖后移除原目标在列表中的条目。
func TestRenameOverwritesExistingFile(t *testing.T) {
	session, dir, items := renameFixture(t)
	batch, err := session.rename([]renameRequest{{items[0].ID, "B.txt"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if batch.CanUndo || batch.Overwritten != 1 || len(batch.RemovedIDs) != 1 || batch.RemovedIDs[0] != items[1].ID {
		t.Fatalf("覆盖结果不正确：%+v", batch)
	}
	assertFile(t, dir, "B.txt", "A.txt")
	if _, err := os.Lstat(filepath.Join(dir, "A.txt")); !os.IsNotExist(err) {
		t.Fatalf("原名称应当已经移除：%v", err)
	}
}

// 批次内目标相同的文件依次覆盖，最终保留列表最后一项。
func TestSameTargetKeepsLastFile(t *testing.T) {
	session, dir, items := renameFixture(t)
	batch, err := session.rename([]renameRequest{{items[0].ID, "C.txt"}, {items[1].ID, "C.txt"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if batch.CanUndo || batch.Overwritten != 1 || len(batch.Changes) != 1 || batch.Changes[0].ID != items[1].ID ||
		len(batch.RemovedIDs) != 1 || batch.RemovedIDs[0] != items[0].ID {
		t.Fatalf("重复目标的执行结果不正确：%+v", batch)
	}
	assertFile(t, dir, "C.txt", "B.txt")
}
