# FFmpeg GUI Converter

一个基于 `Go + Fyne + ffmpeg` 的本地图形界面转换器。

现在的定位不再只是“单一转码器”，而是一个按功能分类整理的轻量媒体全能转换器。

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
- `提取与封装`
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
- 自动探测同目录或系统 `PATH` 中的 `ffmpeg` / `ffprobe`

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

### 提取与封装

- `MP3 / 音轨提取`
- `WAV / 音轨提取`
- `MKV / 直接封装（不转码）`
- `MP4 / 直接封装（不转码）`
- `MOV / 直接封装（不转码）`

## 依赖

开发 / 本地运行需要：

- `Go 1.24+`
- `ffmpeg`
- `ffprobe`

如果只是给最终用户使用，后面可以直接构建出二进制，再把 `ffmpeg` 放到同目录一起分发。

## 运行

### 直接运行

```bash
go run .
```

### 双击启动

```bash
./run.command
```

### 构建本地二进制

```bash
./build.command
```

构建结果默认在：

```text
bin/ffmpeg-gui-converter
```

Windows 下可用：

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

## 目录结构

```text
ffmpeg-gui-converter/
├── main.go
├── go.mod
├── go.sum
├── build.command
├── run.command
├── run.bat
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
- 打包 macOS `.app`
- 打包 Windows `.exe`
