//! IMF Viewer — Tauri native wrapper for the IMF web GUI.
//!
//! Architecture:
//! 1. Launches the Go `imf` binary as a sidecar with `imf gui`
//! 2. Sets IMF_NO_BROWSER=1 so the Go binary doesn't open a browser
//! 3. Detects the port from sidecar stdout
//! 4. Handles macOS file association via RunEvent::Opened (Apple Events)
//! 5. Creates a native Tauri webview window pointing at the local HTTP server
//! 6. Kills the sidecar on window close

use std::io::{BufRead, BufReader};
use std::process::{Child, Command, Stdio};
use std::sync::Mutex;
use tauri::Manager;

struct SidecarState {
    child: Mutex<Option<Child>>,
    port: Mutex<u16>,
}

fn sidecar_path(app: &tauri::AppHandle) -> std::path::PathBuf {
    let binary_name = if cfg!(target_os = "windows") { "imf.exe" } else { "imf" };
    let mut candidates: Vec<std::path::PathBuf> = Vec::new();
    if let Ok(resource_dir) = app.path().resource_dir() {
        candidates.push(resource_dir.join("sidecar").join(binary_name));
        candidates.push(resource_dir.join(binary_name));
    }
    if let Ok(exe) = std::env::current_exe() {
        if let Some(exe_dir) = exe.parent() {
            candidates.push(exe_dir.join(binary_name));
        }
    }
    if let Ok(cwd) = std::env::current_dir() {
        candidates.push(cwd.join(binary_name));
        candidates.push(cwd.join("..").join(binary_name));
    }
    for path in &candidates {
        if path.exists() {
            return path.clone();
        }
    }
    std::path::PathBuf::from(binary_name)
}

fn launch_sidecar(app: &tauri::AppHandle) -> Result<(Child, u16), String> {
    let binary = sidecar_path(app);
    let mut child = Command::new(&binary)
        .arg("gui")
        .env("IMF_NO_BROWSER", "1")
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit())
        .spawn()
        .map_err(|e| format!("Failed to launch sidecar at {:?}: {}", binary, e))?;
    let stdout = child.stdout.take().ok_or("Failed to capture stdout")?;
    let reader = BufReader::new(stdout);
    let mut port: u16 = 0;
    for line in reader.lines().map_while(Result::ok) {
        if line.contains("running at http://127.0.0.1:") {
            if let Some(port_str) = line.rsplit(':').next() {
                if let Ok(p) = port_str.trim().parse::<u16>() {
                    port = p;
                    break;
                }
            }
        }
    }
    if port == 0 {
        let _ = child.kill();
        return Err("Could not detect sidecar port".to_string());
    }
    Ok((child, port))
}

fn urlencod(s: &str) -> String {
    let mut r = String::new();
    for b in s.bytes() {
        match b {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => r.push(b as char),
            _ => r.push_str(&format!("%{:02X}", b)),
        }
    }
    r
}

/// Copy .imf file to sidecar workdir (Desktop). Returns filename.
fn copy_to_workdir(file_path: &str) -> Option<String> {
    let path = std::path::Path::new(file_path);
    let file_name = path.file_name()?.to_string_lossy().to_string();
    if let Some(home) = std::env::var_os("HOME") {
        let desktop = std::path::Path::new(&home).join("Desktop");
        let dest_dir = if desktop.is_dir() { desktop }
            else {
                let dl = std::path::Path::new(&home).join("Downloads");
                if dl.is_dir() { dl } else { std::env::temp_dir() }
            };
        let dest = dest_dir.join(&file_name);
        if path.canonicalize().ok() != dest.canonicalize().ok() {
            let _ = std::fs::copy(file_path, &dest);
        }
    }
    Some(file_name)
}

/// Extract .imf file path from a RunEvent::Opened URL.
fn imf_path_from_url(url: &tauri::Url) -> Option<String> {
    let path_str = if url.scheme() == "file" {
        url.to_file_path().ok().map(|p| p.to_string_lossy().to_string())
    } else {
        Some(url.to_string())
    };
    path_str.filter(|p| p.ends_with(".imf"))
}

fn main() {
    // Shared state to capture file paths from early RunEvent::Opened events.
    // On macOS, Opened fires BEFORE setup() when double-clicking a file to launch.
    let pending = std::sync::Arc::new(Mutex::new(Option::<String>::None));
    let pending_for_setup = pending.clone();

    let app = tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .setup(move |app| {
            let handle = app.handle().clone();
            let (child, port) = launch_sidecar(&handle)
                .map_err(|e| Box::new(std::io::Error::new(std::io::ErrorKind::Other, e)))?;

            app.manage(SidecarState {
                child: Mutex::new(Some(child)),
                port: Mutex::new(port),
            });

            // Check if a file path was stored by an early Opened event
            let pending_file = pending_for_setup.lock().ok().and_then(|mut p| p.take());
            let file_name = pending_file.as_ref().and_then(|path| copy_to_workdir(path));

            let url = match file_name {
                Some(ref name) => format!("http://127.0.0.1:{}/?open={}", port, urlencod(name)),
                None => format!("http://127.0.0.1:{}", port),
            };

            let _window = tauri::WebviewWindowBuilder::new(
                &handle,
                "main",
                tauri::WebviewUrl::External(url.parse().unwrap()),
            )
            .title("IMF Viewer")
            .inner_size(1100.0, 750.0)
            .min_inner_size(800.0, 500.0)
            .center()
            .on_navigation(|url| {
                url.host_str() == Some("127.0.0.1")
                    || url.host_str() == Some("localhost")
                    || url.scheme() == "tauri"
                    || url.scheme() == "about"
            })
            .build()?;

            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::Destroyed = event {
                if let Some(state) = window.try_state::<SidecarState>() {
                    if let Ok(mut child) = state.child.lock() {
                        if let Some(ref mut c) = *child {
                            let _ = c.kill();
                        }
                    }
                }
            }
        })
        .build(tauri::generate_context!())
        .expect("error building IMF Viewer");

    // Handle macOS file association via RunEvent::Opened.
    // When user double-clicks an .imf file, macOS sends an Apple Event
    // which Tauri delivers as RunEvent::Opened with file:// URLs.
    app.run(move |app_handle, event| {
        if let tauri::RunEvent::Opened { urls } = &event {
            for url in urls {
                if let Some(path) = imf_path_from_url(url) {
                    // If sidecar is ready, navigate the existing window
                    if let Some(state) = app_handle.try_state::<SidecarState>() {
                        if let Ok(port) = state.port.lock() {
                            if let Some(file_name) = copy_to_workdir(&path) {
                                let nav_url = format!(
                                    "http://127.0.0.1:{}/?open={}",
                                    *port, urlencod(&file_name)
                                );
                                if let Some(window) = app_handle.get_webview_window("main") {
                                    let _ = window.navigate(nav_url.parse().unwrap());
                                }
                            }
                            return;
                        }
                    }
                    // Sidecar not ready yet — store for setup() to pick up
                    if let Ok(mut p) = pending.lock() {
                        *p = Some(path);
                    }
                }
            }
        }
    });
}
