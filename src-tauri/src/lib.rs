#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    // 仅注册文件选择与文件系统插件，不在 Rust 层承载重命名业务。
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .run(tauri::generate_context!())
        .expect("启动 Rename Studio 失败");
}
