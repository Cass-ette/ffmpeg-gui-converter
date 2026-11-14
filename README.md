# FFmpeg GUI Converter

一个基于 `Go + Fyne + ffmpeg` 的本地图形界面转换器。

现在的定位不再只是“单一转码器”，而是一个按功能分类整理的轻量媒体全能转换器，并且更偏向“构建一次，双击就用”的桌面应用交付方式。

目标很明确：

- 用 `ffmpeg` 做转换核心
- 桌面 GUI，不走 Electron
- 运行时尽量轻
- 后面容易打包成单文件可执行程序

## 为什么改成 Go

相比 `Python + tkinter`，Go 版更适合“下载后直接点开”的交付方式：

- 最终可以编译成单个可执行文件
- 不依赖目标机器上的 Python 环境
- 不会踩 `tkinter` / `_tkinter` 缺失这类解释器差异

当前仓库仍然依赖系统里的 `ffmpeg` / `ffprobe`，但程序本身是 Go GUI 应用。

## 当前功能

- 选择输入文件
- 自动建议输出路径
- 选择输出目录
- 按分类管理转换预设
- `视频转码`
- `音频转码`
- `视频提取音频`
- `视频封装`
- 每个预设都有说明文案
- 分辨率预设
- 画面模式
- `补边适配目标画幅`
- `裁剪填满目标画幅`
- `等比缩放（不强制画幅）`
- 可选设置 FPS
- 可调 `CRF`
- 追加自定义 `ffmpeg` 参数
- 实时进度
- 实时日志
- 转完后直接打开结果位置
- 自动探测程序同目录、`bin/`、macOS `.app` 包内 `Contents/Resources/bin/`，以及系统 `PATH` 中的 `ffmpeg` / `ffprobe`

## 当前预设分类

### 视频转码

- `MP4 / H.264（通用兼容）`
- `MP4 / H.265（更省体积）`
- `WebM / VP9（网页友好）`
- `MOV / ProRes（剪辑中间格式）`

如果你要把横屏素材做成真正的竖版成片：

- 选 `1080x1920` 或 `720x1280` 这类竖屏分辨率
- 画面模式优先用 `补边适配目标画幅`
- 如果想让画面铺满竖屏，再切到 `裁剪填满目标画幅`

### 音频转码

- `MP3 / 通用音频`
- `M4A / AAC`
- `FLAC / 无损音频`
- `WAV / PCM 无损`
- `OGG / Vorbis`
- `Opus / 高压缩`

### 视频提取音频

- `MP3 / 提取视频音轨`
- `WAV / 提取视频音轨`

选择带有音轨的视频作为输入，即可导出 MP3 或 WAV 音频文件。

### 视频封装

- `MKV / 直接封装（不转码）`
- `MP4 / 直接封装（不转码）`
- `MOV / 直接封装（不转码）`

## 依赖

开发 / 本地运行需要：

- `Go 1.24+`
- `ffmpeg`
- `ffprobe`

如果只是给最终用户使用，可以直接构建出二进制 / `.app`，再把 `ffmpeg` / `ffprobe` 一起分发。

## 运行

### 直接运行

```bash
go run .
```

### 双击启动

```bash
./run.command
```

如果已经打过包，它会优先打开已有的 `.app` 或本地二进制。

### 构建当前机器的开发二进制

```bash
./build.command
```

构建结果默认在：

```text
bin/ffmpeg-gui-converter
```

### 打包 macOS `.app`

```bash
./package-mac-app.command
```

输出目录：

```text
dist/macos/FFmpeg GUI Converter.app
```

如果系统里已经能找到 `ffmpeg` / `ffprobe`，打包脚本会自动把它们复制进 `.app/Contents/Resources/bin/`。

想直接安装到“应用程序”目录可执行：

```bash
./install-mac-app.command
```

默认安装到：

```text
/Applications/FFmpeg GUI Converter.app
```

### 构建 Windows GUI `.exe`

在 Windows 机器上：

```bat
build-windows.bat
```

如果你在 macOS / Linux 上交叉编译：

```bash
./build-windows.command
```

这个脚本需要 `mingw-w64` 或 `zig` 作为 Windows 交叉编译工具链。

输出目录：

```text
dist/windows/ffmpeg-gui-converter/
├── ffmpeg-gui-converter.exe
├── README.txt
└── bin/
```

给最终用户分发时，推荐把 `ffmpeg.exe` 和 `ffprobe.exe` 放到：

```text
dist/windows/ffmpeg-gui-converter/bin/
```

或者直接与 `ffmpeg-gui-converter.exe` 放在同一目录。

### Windows 本地双击启动

```bat
run.bat
```

## 安装 ffmpeg

### macOS

```bash
brew install ffmpeg
```

### Linux

使用系统包管理器安装 `ffmpeg`。

### Windows

安装 FFmpeg 后，把 `ffmpeg.exe` 和 `ffprobe.exe` 放进 `PATH`，或者放到程序同目录。

## 推荐分发方式

### macOS

- 构建 `dist/macos/FFmpeg GUI Converter.app`
- 把 `.app` 拖进“应用程序”
- 如果要免安装依赖，确保 `.app/Contents/Resources/bin/` 里有 `ffmpeg` 和 `ffprobe`

### Windows

- 分发整个 `dist/windows/ffmpeg-gui-converter/` 目录
- 让用户直接双击 `ffmpeg-gui-converter.exe`
- 把 `ffmpeg.exe` 和 `ffprobe.exe` 放在同目录或 `bin/` 目录

## 关于 macOS 打开方式

- 本地自己构建的 `.app`，通常可以直接从“应用程序”中打开
- 如果后面要把应用发给别人下载，未签名 / 未公证的 `.app` 仍可能被 Gatekeeper 提示
- 真正面向外部分发时，建议再补应用图标、签名和 notarization

## 目录结构

```text
ffmpeg-gui-converter/
├── assets/
│   └── macos/
│       ├── app-icon-source.png
│       ├── AppIcon-1024.png
│       └── AppIcon.icns
├── main.go
├── go.mod
├── go.sum
├── build.command
├── package-mac-app.command
├── install-mac-app.command
├── build-windows.command
├── build-windows.bat
├── run.command
├── run.bat
├── packaging/
│   ├── macos/
│   │   ├── Info.plist
│   │   └── make_app_icon.py
│   └── windows/
│       └── README.txt
└── README.md
```

## 后续可以继续加

- 批量任务队列
- 拖拽导入
- 裁剪片段
- 水印
- 字幕烧录
- GPU 编码预设
- 记住上次使用的配置
- 应用图标
- macOS / Windows 签名与正式发行包
