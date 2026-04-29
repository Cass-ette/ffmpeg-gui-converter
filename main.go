package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	appTitle             = "FFmpeg GUI Converter"
	defaultPresetKey     = "h264_mp4"
	categoryVideo        = "视频转码"
	categoryAudio        = "音频转码"
	categoryTool         = "提取与封装"
	defaultResolution    = "保持原始"
	defaultFrameMode     = "补边适配目标画幅（推荐）"
	defaultWindowWidth   = 1040
	defaultWindowHeight  = 860
	maxLogLines          = 1500
	defaultFFmpegCRF     = "23"
	defaultMediaSummary  = "媒体信息：选择文件后自动读取"
	defaultPreviewPrompt = "选择输入文件后，这里会显示即将执行的 ffmpeg 命令。"
)

type Preset struct {
	Key                string
	Category           string
	Label              string
	Summary            string
	Extension          string
	Args               []string
	SupportsCRF        bool
	AudioOnly          bool
	AllowsVideoFilters bool
}

type ResolutionOption struct {
	Label  string
	Width  int
	Height int
}

type ProbeResult struct {
	DurationSeconds float64
	Summary         string
}

type FrameModeOption struct {
	Label string
	Key   string
}

type ConversionRequest struct {
	InputPath       string
	OutputPath      string
	Preset          Preset
	Resolution      ResolutionOption
	FrameMode       string
	FPS             float64
	CRF             int
	ExtraArgs       []string
	DurationSeconds float64
}

type FFProbePayload struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Channels  int    `json:"channels"`
	} `json:"streams"`
}

type ConverterUI struct {
	app    fyne.App
	window fyne.Window

	ffmpegPath  string
	ffprobePath string

	inputEntry       *widget.Entry
	outputEntry      *widget.Entry
	fpsEntry         *widget.Entry
	crfEntry         *widget.Entry
	extraArgsEntry   *widget.Entry
	categorySelect   *widget.Select
	presetSelect     *widget.Select
	resolutionSelect *widget.Select
	frameModeSelect  *widget.Select

	statusLabel     *widget.Label
	percentLabel    *widget.Label
	elapsedLabel    *widget.Label
	dependencyLabel *widget.Label
	mediaLabel      *widget.Label
	presetHintLabel *widget.Label

	progressBar      *widget.ProgressBar
	progressInfinite *widget.ProgressBarInfinite
	previewEntry     *widget.Entry
	logEntry         *widget.Entry

	startButton  *widget.Button
	stopButton   *widget.Button
	revealButton *widget.Button

	lastSuggestedOutput string
	lastSuccessOutput   string
	lastDuration        float64

	timerStop chan struct{}

	logLines []string

	procMu        sync.Mutex
	currentCmd    *exec.Cmd
	stopRequested bool
	startedAt     time.Time
	initializing  bool
}

var presets = []Preset{
	{
		Key:       "h264_mp4",
		Category:  categoryVideo,
		Label:     "MP4 / H.264（通用兼容）",
		Summary:   "最通用的视频输出格式，适合播放器、网盘和大多数社交平台。",
		Extension: "mp4",
		Args: []string{
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "{crf}",
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
			"-c:a", "aac",
			"-b:a", "192k",
		},
		SupportsCRF:        true,
		AllowsVideoFilters: true,
	},
	{
		Key:       "h265_mp4",
		Category:  categoryVideo,
		Label:     "MP4 / H.265（更省体积）",
		Summary:   "通常体积更小，但老设备和老播放器兼容性略差于 H.264。",
		Extension: "mp4",
		Args: []string{
			"-c:v", "libx265",
			"-preset", "medium",
			"-crf", "{crf}",
			"-tag:v", "hvc1",
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
			"-c:a", "aac",
			"-b:a", "160k",
		},
		SupportsCRF:        true,
		AllowsVideoFilters: true,
	},
	{
		Key:       "vp9_webm",
		Category:  categoryVideo,
		Label:     "WebM / VP9（网页友好）",
		Summary:   "适合网页发布和较小体积需求，现代浏览器兼容性较好。",
		Extension: "webm",
		Args: []string{
			"-c:v", "libvpx-vp9",
			"-b:v", "0",
			"-crf", "{crf}",
			"-c:a", "libopus",
			"-b:a", "128k",
		},
		SupportsCRF:        true,
		AllowsVideoFilters: true,
	},
	{
		Key:       "prores_mov",
		Category:  categoryVideo,
		Label:     "MOV / ProRes（剪辑中间格式）",
		Summary:   "适合后期剪辑流程，文件会明显更大，但编辑友好。",
		Extension: "mov",
		Args: []string{
			"-c:v", "prores_ks",
			"-profile:v", "3",
			"-pix_fmt", "yuv422p10le",
			"-c:a", "pcm_s16le",
		},
		AllowsVideoFilters: true,
	},
	{
		Key:       "audio_mp3",
		Category:  categoryAudio,
		Label:     "MP3 / 通用音频",
		Summary:   "最通用的音频输出，兼容性最好，适合车机、手机和普通播放器。",
		Extension: "mp3",
		Args:      []string{"-vn", "-c:a", "libmp3lame", "-q:a", "2"},
		AudioOnly: true,
	},
	{
		Key:       "audio_m4a",
		Category:  categoryAudio,
		Label:     "M4A / AAC",
		Summary:   "适合日常听音和移动设备，体积与音质比较均衡。",
		Extension: "m4a",
		Args: []string{
			"-vn",
			"-c:a", "aac",
			"-b:a", "192k",
		},
		AudioOnly: true,
	},
	{
		Key:       "audio_flac",
		Category:  categoryAudio,
		Label:     "FLAC / 无损音频",
		Summary:   "保留无损音质，适合归档和高质量收藏。",
		Extension: "flac",
		Args:      []string{"-vn", "-c:a", "flac"},
		AudioOnly: true,
	},
	{
		Key:       "audio_wav",
		Category:  categoryAudio,
		Label:     "WAV / PCM 无损",
		Summary:   "兼容性强的无损 PCM 输出，适合进一步音频编辑。",
		Extension: "wav",
		Args:      []string{"-vn", "-c:a", "pcm_s16le"},
		AudioOnly: true,
	},
	{
		Key:       "audio_ogg",
		Category:  categoryAudio,
		Label:     "OGG / Vorbis",
		Summary:   "开源音频格式，适合通用压缩音频场景。",
		Extension: "ogg",
		Args: []string{
			"-vn",
			"-c:a", "libvorbis",
			"-q:a", "5",
		},
		AudioOnly: true,
	},
	{
		Key:       "audio_opus",
		Category:  categoryAudio,
		Label:     "Opus / 高压缩",
		Summary:   "在较低码率下通常有不错表现，适合语音和轻量发布。",
		Extension: "opus",
		Args: []string{
			"-vn",
			"-c:a", "libopus",
			"-b:a", "128k",
		},
		AudioOnly: true,
	},
	{
		Key:       "extract_mp3",
		Category:  categoryTool,
		Label:     "MP3 / 音轨提取",
		Summary:   "从视频里抽出音轨并转成 MP3，适合课程、播客、访谈。",
		Extension: "mp3",
		Args:      []string{"-vn", "-c:a", "libmp3lame", "-q:a", "2"},
		AudioOnly: true,
	},
	{
		Key:       "extract_wav",
		Category:  categoryTool,
		Label:     "WAV / 音轨提取",
		Summary:   "从视频里抽出无损音轨，适合后续剪辑和音频处理。",
		Extension: "wav",
		Args:      []string{"-vn", "-c:a", "pcm_s16le"},
		AudioOnly: true,
	},
	{
		Key:       "copy_mkv",
		Category:  categoryTool,
		Label:     "MKV / 直接封装（不转码）",
		Summary:   "只改封装不改编码，速度最快；适合已有编码本身就可用的文件。",
		Extension: "mkv",
		Args:      []string{"-c", "copy"},
	},
	{
		Key:       "copy_mp4",
		Category:  categoryTool,
		Label:     "MP4 / 直接封装（不转码）",
		Summary:   "只改封装不改编码，适合本身就兼容 MP4 的视频和音频流。",
		Extension: "mp4",
		Args: []string{
			"-c", "copy",
			"-movflags", "+faststart",
		},
	},
	{
		Key:       "copy_mov",
		Category:  categoryTool,
		Label:     "MOV / 直接封装（不转码）",
		Summary:   "适合把已有流快速装进 MOV 容器，常用于剪辑工作流交接。",
		Extension: "mov",
		Args:      []string{"-c", "copy"},
	},
}

var resolutionOptions = []ResolutionOption{
	{Label: defaultResolution},
	{Label: "1920x1080 (1080p 横屏)", Width: 1920, Height: 1080},
	{Label: "1280x720 (720p 横屏)", Width: 1280, Height: 720},
	{Label: "854x480 (480p 横屏)", Width: 854, Height: 480},
	{Label: "1080x1920 (1080p 竖屏)", Width: 1080, Height: 1920},
	{Label: "720x1280 (720p 竖屏)", Width: 720, Height: 1280},
}

var frameModeOptions = []FrameModeOption{
	{Label: defaultFrameMode, Key: "pad"},
	{Label: "裁剪填满目标画幅", Key: "crop"},
	{Label: "等比缩放（不强制画幅）", Key: "fit"},
}

func main() {
	application := app.NewWithID("dev.chenzilve.ffmpeg-gui-converter")
	window := application.NewWindow(appTitle)
	window.Resize(fyne.NewSize(defaultWindowWidth, defaultWindowHeight))

	ui := newConverterUI(application, window)
	window.SetContent(ui.buildContent())
	window.SetCloseIntercept(func() {
		if ui.isRunning() {
			dialog.ShowConfirm(
				"任务仍在运行",
				"当前正在转换，确定关闭应用吗？",
				func(ok bool) {
					if !ok {
						return
					}
					window.SetCloseIntercept(nil)
					ui.stopConversion()
					window.Close()
				},
				window,
			)
			return
		}
		window.SetCloseIntercept(nil)
		window.Close()
	})

	window.ShowAndRun()
}

func newConverterUI(application fyne.App, window fyne.Window) *ConverterUI {
	ui := &ConverterUI{
		app:          application,
		window:       window,
		ffmpegPath:   detectBinary("ffmpeg"),
		ffprobePath:  detectBinary("ffprobe"),
		lastDuration: 0,
	}
	return ui
}

func (ui *ConverterUI) buildContent() fyne.CanvasObject {
	ui.initializing = true

	ui.inputEntry = widget.NewEntry()
	ui.inputEntry.SetPlaceHolder("选择或粘贴输入媒体文件路径")
	ui.inputEntry.OnChanged = func(string) {
		ui.handleInputChanged()
	}

	ui.outputEntry = widget.NewEntry()
	ui.outputEntry.SetPlaceHolder("输出文件路径")
	ui.outputEntry.OnChanged = func(string) {
		ui.refreshPreview()
	}

	ui.fpsEntry = widget.NewEntry()
	ui.fpsEntry.SetPlaceHolder("例如 30 / 29.97")
	ui.fpsEntry.OnChanged = func(string) {
		ui.refreshPreview()
	}

	ui.crfEntry = widget.NewEntry()
	ui.crfEntry.SetPlaceHolder("0 - 51")
	ui.crfEntry.SetText(defaultFFmpegCRF)
	ui.crfEntry.OnChanged = func(string) {
		ui.refreshPreview()
	}

	ui.extraArgsEntry = widget.NewEntry()
	ui.extraArgsEntry.SetPlaceHolder(`例如 -ss 00:00:05 -to 00:00:20 或 -metadata title="Demo"`)
	ui.extraArgsEntry.OnChanged = func(string) {
		ui.refreshPreview()
	}

	ui.categorySelect = widget.NewSelect(categoryLabels(), func(string) {
		if ui.initializing {
			return
		}
		ui.refreshPresetOptions()
	})

	ui.presetSelect = widget.NewSelect(nil, func(string) {
		if ui.initializing {
			return
		}
		ui.syncPresetControls()
		ui.maybeUpdateSuggestedOutput()
		ui.refreshPresetHint()
		ui.refreshPreview()
	})

	ui.resolutionSelect = widget.NewSelect(resolutionLabels(), func(string) {
		if ui.initializing {
			return
		}
		ui.refreshPreview()
	})

	ui.frameModeSelect = widget.NewSelect(frameModeLabels(), func(string) {
		if ui.initializing {
			return
		}
		ui.refreshPreview()
	})

	ui.statusLabel = widget.NewLabel("状态：就绪")
	ui.percentLabel = widget.NewLabel("0%")
	ui.elapsedLabel = widget.NewLabel("已耗时 00:00:00")
	ui.dependencyLabel = widget.NewLabel(ui.dependencySummary())
	ui.dependencyLabel.Wrapping = fyne.TextWrapWord
	ui.mediaLabel = widget.NewLabel(defaultMediaSummary)
	ui.mediaLabel.Wrapping = fyne.TextWrapWord
	ui.presetHintLabel = widget.NewLabel("")
	ui.presetHintLabel.Wrapping = fyne.TextWrapWord

	ui.progressBar = widget.NewProgressBar()
	ui.progressBar.Min = 0
	ui.progressBar.Max = 1
	ui.progressInfinite = widget.NewProgressBarInfinite()
	ui.progressInfinite.Hide()

	ui.previewEntry = widget.NewMultiLineEntry()
	ui.previewEntry.SetText(defaultPreviewPrompt)
	ui.previewEntry.Wrapping = fyne.TextWrapWord
	ui.previewEntry.SetMinRowsVisible(4)
	ui.previewEntry.Disable()

	ui.logEntry = widget.NewMultiLineEntry()
	ui.logEntry.SetText("[启动] 应用已就绪。\n")
	ui.logEntry.Wrapping = fyne.TextWrapWord
	ui.logEntry.SetMinRowsVisible(14)
	ui.logEntry.Disable()

	ui.startButton = widget.NewButtonWithIcon("开始转换", theme.MediaPlayIcon(), func() {
		ui.startConversion()
	})
	ui.startButton.Importance = widget.HighImportance

	ui.stopButton = widget.NewButtonWithIcon("停止", theme.MediaStopIcon(), func() {
		ui.stopConversion()
	})
	ui.stopButton.Disable()

	ui.revealButton = widget.NewButtonWithIcon("打开结果", theme.FolderOpenIcon(), func() {
		ui.revealOutput()
	})
	ui.revealButton.Disable()

	ui.categorySelect.SetSelected(defaultPreset().Category)
	ui.presetSelect.SetOptions(presetLabelsForCategory(defaultPreset().Category))
	ui.presetSelect.SetSelected(defaultPreset().Label)
	ui.resolutionSelect.SetSelected(defaultResolution)
	ui.frameModeSelect.SetSelected(defaultFrameMode)
	ui.initializing = false
	ui.syncPresetControls()
	ui.refreshPresetHint()
	ui.refreshPreview()

	if ui.ffmpegPath == "" {
		ui.startButton.Disable()
		ui.statusLabel.SetText("状态：未找到 ffmpeg，无法开始转换")
	}

	headerTitle := widget.NewLabelWithStyle(appTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	headerSubtitle := widget.NewLabel("按分类整理的视频、音频、提取与封装转换器。优先查找程序同目录下的 ffmpeg / ffprobe，其次回退系统 PATH。")
	headerSubtitle.Wrapping = fyne.TextWrapWord
	header := widget.NewCard("", "", container.NewVBox(headerTitle, headerSubtitle))

	inputRow := container.NewBorder(
		nil,
		nil,
		nil,
		widget.NewButtonWithIcon("选择文件", theme.FolderOpenIcon(), ui.chooseInputFile),
		ui.inputEntry,
	)
	outputButtons := container.NewHBox(
		widget.NewButton("建议路径", ui.applySuggestedOutput),
		widget.NewButton("选择目录", ui.chooseOutputDirectory),
	)
	outputRow := container.NewBorder(nil, nil, nil, outputButtons, ui.outputEntry)

	fileForm := container.New(layout.NewFormLayout(),
		widget.NewLabel("输入文件"), inputRow,
		widget.NewLabel("输出文件"), outputRow,
		widget.NewLabel("功能分类"), ui.categorySelect,
		widget.NewLabel("转换预设"), ui.presetSelect,
		widget.NewLabel("预设说明"), ui.presetHintLabel,
		widget.NewLabel("分辨率"), ui.resolutionSelect,
		widget.NewLabel("画面模式"), ui.frameModeSelect,
		widget.NewLabel("FPS"), ui.fpsEntry,
		widget.NewLabel("CRF"), ui.crfEntry,
		widget.NewLabel("额外参数"), ui.extraArgsEntry,
		widget.NewLabel("依赖检测"), ui.dependencyLabel,
		widget.NewLabel("媒体信息"), ui.mediaLabel,
	)

	fileCard := widget.NewCard("文件与参数", "轻量方案，直接调用 ffmpeg / ffprobe。", fileForm)

	buttonRow := container.NewBorder(
		nil,
		nil,
		container.NewHBox(ui.startButton, ui.stopButton, ui.revealButton),
		nil,
		ui.statusLabel,
	)

	progressBarStack := container.NewStack(ui.progressBar, ui.progressInfinite)
	progressMeta := container.NewBorder(nil, nil, ui.percentLabel, ui.elapsedLabel, nil)
	progressCard := widget.NewCard("进度", "", container.NewVBox(progressBarStack, progressMeta))

	previewCard := widget.NewCard("命令预览", "", ui.previewEntry)
	logCard := widget.NewCard("运行日志", "", ui.logEntry)

	content := container.NewVBox(
		header,
		fileCard,
		buttonRow,
		progressCard,
		previewCard,
		logCard,
	)

	return container.NewPadded(container.NewVScroll(content))
}

func (ui *ConverterUI) dependencySummary() string {
	ffmpegSummary := ui.ffmpegPath
	if ffmpegSummary == "" {
		ffmpegSummary = "未找到"
	}
	ffprobeSummary := ui.ffprobePath
	if ffprobeSummary == "" {
		ffprobeSummary = "未找到"
	}
	return fmt.Sprintf("ffmpeg：%s    ffprobe：%s", ffmpegSummary, ffprobeSummary)
}

func (ui *ConverterUI) handleInputChanged() {
	ui.maybeUpdateSuggestedOutput()
	ui.refreshMediaSummary()
	ui.refreshPreview()
}

func (ui *ConverterUI) refreshPresetOptions() {
	category := ui.selectedCategory()
	options := presetLabelsForCategory(category)
	if len(options) == 0 {
		return
	}

	current := ui.presetSelect.Selected
	ui.initializing = true
	ui.presetSelect.SetOptions(options)
	if !containsString(options, current) {
		current = options[0]
	}
	ui.presetSelect.SetSelected(current)
	ui.initializing = false

	ui.syncPresetControls()
	ui.maybeUpdateSuggestedOutput()
	ui.refreshPresetHint()
	ui.refreshPreview()
}

func (ui *ConverterUI) refreshPresetHint() {
	preset := ui.selectedPreset()
	ui.presetHintLabel.SetText(preset.Summary)
}

func (ui *ConverterUI) syncPresetControls() {
	preset := ui.selectedPreset()
	if preset.SupportsCRF {
		ui.crfEntry.Enable()
	} else {
		ui.crfEntry.Disable()
	}

	if preset.AllowsVideoFilters {
		ui.resolutionSelect.Enable()
		ui.frameModeSelect.Enable()
		ui.fpsEntry.Enable()
	} else {
		ui.resolutionSelect.Disable()
		ui.frameModeSelect.Disable()
		ui.fpsEntry.Disable()
		ui.resolutionSelect.SetSelected(defaultResolution)
		ui.frameModeSelect.SetSelected(defaultFrameMode)
		ui.fpsEntry.SetText("")
	}
}

func (ui *ConverterUI) chooseInputFile() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, ui.window)
			return
		}
		if reader == nil {
			return
		}
		path := reader.URI().Path()
		_ = reader.Close()
		ui.inputEntry.SetText(path)
	}, ui.window)
	fileDialog.SetTitleText("选择输入文件")
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{
		".mp4", ".mov", ".mkv", ".avi", ".webm", ".m4v", ".flv", ".ts", ".mts", ".m2ts", ".mpg", ".mpeg", ".3gp",
		".mp3", ".wav", ".m4a", ".flac", ".aac", ".ogg", ".opus", ".wma", ".ape",
	}))
	if dir := ui.bestInitialDirectory(); dir != "" {
		if listable, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
			fileDialog.SetLocation(listable)
		}
	}
	fileDialog.Show()
}

func (ui *ConverterUI) chooseOutputDirectory() {
	folderDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, ui.window)
			return
		}
		if uri == nil {
			return
		}
		filename := ui.currentOutputFileName()
		ui.outputEntry.SetText(filepath.Join(uri.Path(), filename))
	}, ui.window)
	folderDialog.SetTitleText("选择输出目录")
	if dir := ui.bestInitialDirectory(); dir != "" {
		if listable, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
			folderDialog.SetLocation(listable)
		}
	}
	folderDialog.Show()
}

func (ui *ConverterUI) bestInitialDirectory() string {
	for _, candidate := range []string{
		filepath.Dir(strings.TrimSpace(ui.inputEntry.Text)),
		filepath.Dir(strings.TrimSpace(ui.outputEntry.Text)),
		userHomeDir(),
	} {
		if candidate == "" || candidate == "." {
			continue
		}
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func (ui *ConverterUI) currentOutputFileName() string {
	current := strings.TrimSpace(ui.outputEntry.Text)
	if current != "" {
		base := filepath.Base(current)
		if base != "." && base != string(filepath.Separator) {
			return base
		}
	}

	suggested := ui.suggestedOutputPath()
	if suggested != "" {
		return filepath.Base(suggested)
	}

	return "output." + ui.selectedPreset().Extension
}

func (ui *ConverterUI) applySuggestedOutput() {
	suggested := ui.suggestedOutputPath()
	if suggested == "" {
		return
	}
	ui.lastSuggestedOutput = suggested
	ui.outputEntry.SetText(suggested)
}

func (ui *ConverterUI) maybeUpdateSuggestedOutput() {
	suggested := ui.suggestedOutputPath()
	if suggested == "" {
		return
	}
	current := strings.TrimSpace(ui.outputEntry.Text)
	if current == "" || current == ui.lastSuggestedOutput {
		ui.lastSuggestedOutput = suggested
		ui.outputEntry.SetText(suggested)
	}
}

func (ui *ConverterUI) suggestedOutputPath() string {
	inputPath := strings.TrimSpace(ui.inputEntry.Text)
	if inputPath == "" {
		return ""
	}
	ext := ui.selectedPreset().Extension
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	dir := filepath.Dir(inputPath)
	return filepath.Join(dir, base+"_converted."+ext)
}

func (ui *ConverterUI) refreshMediaSummary() {
	inputPath := strings.TrimSpace(ui.inputEntry.Text)
	if inputPath == "" {
		ui.mediaLabel.SetText(defaultMediaSummary)
		ui.lastDuration = 0
		return
	}
	if !fileExists(inputPath) {
		ui.mediaLabel.SetText("媒体信息：输入文件不存在")
		ui.lastDuration = 0
		return
	}
	result := ui.probeMedia(inputPath)
	ui.lastDuration = result.DurationSeconds
	ui.mediaLabel.SetText(result.Summary)
}

func (ui *ConverterUI) probeMedia(inputPath string) ProbeResult {
	if ui.ffprobePath == "" {
		return ProbeResult{
			DurationSeconds: 0,
			Summary:         "媒体信息：未找到 ffprobe，无法读取详细信息",
		}
	}

	cmd := exec.Command(
		ui.ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration:stream=index,codec_type,codec_name,width,height,channels",
		"-of", "json",
		inputPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return ProbeResult{
			DurationSeconds: 0,
			Summary:         "媒体信息：读取失败，仍可直接尝试转换",
		}
	}

	var payload FFProbePayload
	if err := json.Unmarshal(output, &payload); err != nil {
		return ProbeResult{
			DurationSeconds: 0,
			Summary:         "媒体信息：读取失败，仍可直接尝试转换",
		}
	}

	duration := parsePositiveFloat(payload.Format.Duration)
	parts := []string{fmt.Sprintf("时长 %s", formatDuration(duration))}

	for _, stream := range payload.Streams {
		switch stream.CodecType {
		case "video":
			parts = append(parts, fmt.Sprintf("视频 %s %dx%d", safeCodec(stream.CodecName), stream.Width, stream.Height))
		case "audio":
			parts = append(parts, fmt.Sprintf("音频 %s %dch", safeCodec(stream.CodecName), stream.Channels))
		}
	}

	return ProbeResult{
		DurationSeconds: duration,
		Summary:         "媒体信息：" + strings.Join(parts, " | "),
	}
}

func (ui *ConverterUI) refreshPreview() {
	request, err := ui.previewRequest()
	if err != nil {
		ui.previewEntry.Enable()
		ui.previewEntry.SetText(defaultPreviewPrompt)
		ui.previewEntry.Disable()
		return
	}
	command := append([]string{ui.ffmpegPathOrFallback()}, ui.buildFFmpegArgs(request)...)
	ui.previewEntry.Enable()
	ui.previewEntry.SetText(renderCommand(command))
	ui.previewEntry.Disable()
}

func (ui *ConverterUI) previewRequest() (ConversionRequest, error) {
	inputPath := strings.TrimSpace(ui.inputEntry.Text)
	outputPath := strings.TrimSpace(ui.outputEntry.Text)
	if inputPath == "" || outputPath == "" || !fileExists(inputPath) {
		return ConversionRequest{}, errors.New("missing paths")
	}
	preset := ui.selectedPreset()
	resolution := resolutionOptionByLabel(ui.resolutionSelect.Selected)
	frameMode := frameModeKeyByLabel(ui.frameModeSelect.Selected)
	if !preset.AllowsVideoFilters {
		resolution = resolutionOptionByLabel(defaultResolution)
		frameMode = ""
	}
	fps := 0.0
	if preset.AllowsVideoFilters {
		fps = parsePositiveFloat(strings.TrimSpace(ui.fpsEntry.Text))
	}
	crf := 23
	if preset.SupportsCRF {
		crf = int(parsePositiveFloat(strings.TrimSpace(ui.crfEntry.Text)))
	}
	extraArgs, err := splitArgs(strings.TrimSpace(ui.extraArgsEntry.Text))
	if err != nil {
		return ConversionRequest{}, err
	}
	return ConversionRequest{
		InputPath:       inputPath,
		OutputPath:      outputPath,
		Preset:          preset,
		Resolution:      resolution,
		FrameMode:       frameMode,
		FPS:             fps,
		CRF:             crf,
		ExtraArgs:       extraArgs,
		DurationSeconds: ui.lastDuration,
	}, nil
}

func (ui *ConverterUI) startConversion() {
	if ui.isRunning() {
		dialog.ShowInformation("任务进行中", "当前已有转换任务在运行。", ui.window)
		return
	}

	request, err := ui.validateRequest()
	if err != nil {
		dialog.ShowError(err, ui.window)
		return
	}

	args := ui.buildFFmpegArgs(request)
	ui.prepareRunUI(request, args)

	go ui.runFFmpeg(request, args)
}

func (ui *ConverterUI) validateRequest() (ConversionRequest, error) {
	if ui.ffmpegPath == "" {
		return ConversionRequest{}, errors.New("没有检测到 ffmpeg，请先安装或把 ffmpeg 放到程序同目录")
	}

	inputPath := strings.TrimSpace(ui.inputEntry.Text)
	outputPath := strings.TrimSpace(ui.outputEntry.Text)

	if inputPath == "" {
		return ConversionRequest{}, errors.New("请先选择输入文件")
	}
	if outputPath == "" {
		return ConversionRequest{}, errors.New("请先设置输出文件")
	}
	if !fileExists(inputPath) {
		return ConversionRequest{}, fmt.Errorf("输入文件不存在：%s", inputPath)
	}

	inputAbs, _ := filepath.Abs(inputPath)
	outputAbs, _ := filepath.Abs(outputPath)
	if filepath.Clean(inputAbs) == filepath.Clean(outputAbs) {
		return ConversionRequest{}, errors.New("输出文件不能与输入文件相同")
	}

	preset := ui.selectedPreset()
	if filepath.Ext(outputPath) == "" {
		outputPath += "." + preset.Extension
		ui.outputEntry.SetText(outputPath)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return ConversionRequest{}, fmt.Errorf("创建输出目录失败：%w", err)
	}

	resolution := resolutionOptionByLabel(ui.resolutionSelect.Selected)
	frameMode := frameModeKeyByLabel(ui.frameModeSelect.Selected)
	if !preset.AllowsVideoFilters {
		resolution = resolutionOptionByLabel(defaultResolution)
		frameMode = ""
	}

	fps := 0.0
	fpsText := strings.TrimSpace(ui.fpsEntry.Text)
	if preset.AllowsVideoFilters && fpsText != "" {
		value, err := strconv.ParseFloat(fpsText, 64)
		if err != nil || value <= 0 {
			return ConversionRequest{}, errors.New("FPS 必须是大于 0 的数字")
		}
		fps = value
	}

	crf := 23
	if preset.SupportsCRF {
		value, err := strconv.Atoi(strings.TrimSpace(ui.crfEntry.Text))
		if err != nil || value < 0 || value > 51 {
			return ConversionRequest{}, errors.New("CRF 必须是 0 到 51 之间的整数")
		}
		crf = value
	}

	extraArgs, err := splitArgs(strings.TrimSpace(ui.extraArgsEntry.Text))
	if err != nil {
		return ConversionRequest{}, fmt.Errorf("额外参数格式错误：%w", err)
	}

	request := ConversionRequest{
		InputPath:       inputPath,
		OutputPath:      outputPath,
		Preset:          preset,
		Resolution:      resolution,
		FrameMode:       frameMode,
		FPS:             fps,
		CRF:             crf,
		ExtraArgs:       extraArgs,
		DurationSeconds: ui.lastDuration,
	}

	if ui.ffprobePath != "" {
		request.DurationSeconds = ui.probeMedia(inputPath).DurationSeconds
	}

	return request, nil
}

func (ui *ConverterUI) prepareRunUI(request ConversionRequest, args []string) {
	ui.lastSuccessOutput = ""
	ui.revealButton.Disable()
	ui.startButton.Disable()
	ui.stopButton.Enable()
	ui.statusLabel.SetText("状态：正在转换")
	ui.percentLabel.SetText("0%")
	ui.elapsedLabel.SetText("已耗时 00:00:00")
	ui.appendLogNow("")
	ui.appendLogNow(strings.Repeat("=", 72))
	ui.appendLogNow(fmt.Sprintf("开始任务：%s -> %s", filepath.Base(request.InputPath), filepath.Base(request.OutputPath)))
	ui.appendLogNow(renderCommand(append([]string{ui.ffmpegPathOrFallback()}, args...)))

	ui.progressBar.SetValue(0)
	ui.progressBar.Show()
	ui.progressInfinite.Stop()
	ui.progressInfinite.Hide()
	if request.DurationSeconds <= 0 {
		ui.progressBar.Hide()
		ui.progressInfinite.Show()
		ui.progressInfinite.Start()
		ui.percentLabel.SetText("等待时长信息")
	}

	ui.setRunningCommand(nil, false)
	ui.startTimer()
}

func (ui *ConverterUI) runFFmpeg(request ConversionRequest, args []string) {
	cmd := exec.Command(ui.ffmpegPathOrFallback(), args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ui.finishWithError(fmt.Errorf("创建 stdout 管道失败：%w", err))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		ui.finishWithError(fmt.Errorf("创建 stderr 管道失败：%w", err))
		return
	}

	if err := cmd.Start(); err != nil {
		ui.finishWithError(fmt.Errorf("启动 ffmpeg 失败：%w", err))
		return
	}

	ui.setRunningCommand(cmd, false)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ui.readProgress(stdout, request.DurationSeconds)
	}()
	go func() {
		defer wg.Done()
		ui.readLogs(stderr)
	}()

	waitErr := cmd.Wait()
	wg.Wait()
	state := ui.runningState()
	stopped := state.stopRequested
	startedAt := state.startedAt
	ui.setRunningCommand(nil, false)

	fyne.Do(func() {
		ui.stopTimer()
		ui.progressInfinite.Stop()
		ui.progressInfinite.Hide()
		ui.progressBar.Show()
		ui.stopButton.Disable()
		if ui.ffmpegPath != "" {
			ui.startButton.Enable()
		}

		if stopped {
			ui.statusLabel.SetText("状态：已停止")
			ui.appendLogNow("任务已停止。")
			return
		}

		if waitErr == nil {
			ui.statusLabel.SetText("状态：转换完成")
			ui.progressBar.SetValue(1)
			ui.percentLabel.SetText("100%")
			if !startedAt.IsZero() {
				ui.elapsedLabel.SetText("已耗时 " + formatDuration(time.Since(startedAt).Seconds()))
			}
			ui.lastSuccessOutput = request.OutputPath
			ui.revealButton.Enable()
			ui.appendLogNow("任务成功完成。")
			ui.appendLogNow("输出文件：" + request.OutputPath)
			return
		}

		ui.statusLabel.SetText("状态：转换失败")
		ui.appendLogNow("任务失败：" + waitErr.Error())
		dialog.ShowError(fmt.Errorf("ffmpeg 执行失败：%w", waitErr), ui.window)
	})
}

func (ui *ConverterUI) readProgress(stdoutPipe ioReadCloser, duration float64) {
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.Contains(line, "=") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		switch key {
		case "out_time":
			seconds := parseFFmpegTimestamp(value)
			if seconds <= 0 {
				continue
			}
			fyne.Do(func() {
				ui.updateProgress(seconds, duration)
			})
		case "progress":
			if value == "end" {
				ui.appendLog("编码完成，正在收尾。")
			}
		}
	}
}

func (ui *ConverterUI) readLogs(stderrPipe ioReadCloser) {
	scanner := bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		ui.appendLog(line)
	}
}

func (ui *ConverterUI) stopConversion() {
	state := ui.runningState()
	if state.cmd == nil {
		return
	}
	ui.statusLabel.SetText("状态：正在停止")
	ui.appendLogNow("收到停止请求，正在终止 ffmpeg ...")
	ui.setRunningCommand(state.cmd, true)
	_ = state.cmd.Process.Kill()
}

func (ui *ConverterUI) revealOutput() {
	if ui.lastSuccessOutput == "" {
		return
	}
	if err := revealInFileManager(ui.lastSuccessOutput); err != nil {
		dialog.ShowError(err, ui.window)
	}
}

func (ui *ConverterUI) updateProgress(processedSeconds, duration float64) {
	if duration > 0 {
		ratio := processedSeconds / duration
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		ui.progressBar.SetValue(ratio)
		ui.percentLabel.SetText(fmt.Sprintf("%.1f%%", ratio*100))
		ui.statusLabel.SetText(fmt.Sprintf("状态：正在转换 %s / %s", formatDuration(processedSeconds), formatDuration(duration)))
		return
	}
	ui.percentLabel.SetText("已处理 " + formatDuration(processedSeconds))
	ui.statusLabel.SetText("状态：正在转换")
}

func (ui *ConverterUI) buildFFmpegArgs(request ConversionRequest) []string {
	args := []string{
		"-hide_banner",
		"-y",
		"-progress", "pipe:1",
		"-nostats",
		"-i", request.InputPath,
	}

	if request.Preset.AllowsVideoFilters {
		filters := make([]string, 0, 4)
		if request.Resolution.Width > 0 && request.Resolution.Height > 0 {
			switch request.FrameMode {
			case "crop":
				filters = append(
					filters,
					fmt.Sprintf(
						"scale=w=%d:h=%d:force_original_aspect_ratio=increase:flags=lanczos",
						request.Resolution.Width,
						request.Resolution.Height,
					),
				)
				filters = append(filters, fmt.Sprintf("crop=%d:%d", request.Resolution.Width, request.Resolution.Height))
			case "fit":
				filters = append(
					filters,
					fmt.Sprintf(
						"scale=w=%d:h=%d:force_original_aspect_ratio=decrease:flags=lanczos",
						request.Resolution.Width,
						request.Resolution.Height,
					),
				)
			default:
				filters = append(
					filters,
					fmt.Sprintf(
						"scale=w=%d:h=%d:force_original_aspect_ratio=decrease:flags=lanczos",
						request.Resolution.Width,
						request.Resolution.Height,
					),
				)
				filters = append(
					filters,
					fmt.Sprintf("pad=%d:%d:(ow-iw)/2:(oh-ih)/2", request.Resolution.Width, request.Resolution.Height),
				)
			}
		}
		if request.FPS > 0 {
			filters = append(filters, fmt.Sprintf("fps=%g", request.FPS))
		}
		if len(filters) > 0 {
			args = append(args, "-vf", strings.Join(filters, ","))
		}
	}

	args = append(args, renderPresetArgs(request.Preset, request.CRF)...)
	args = append(args, request.ExtraArgs...)
	args = append(args, request.OutputPath)
	return args
}

func (ui *ConverterUI) startTimer() {
	ui.stopTimer()
	ui.procMu.Lock()
	ui.startedAt = time.Now()
	ui.procMu.Unlock()
	stop := make(chan struct{})
	ui.timerStop = stop

	go func(ch chan struct{}) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				state := ui.runningState()
				if state.startedAt.IsZero() {
					continue
				}
				elapsed := time.Since(state.startedAt).Seconds()
				fyne.Do(func() {
					ui.elapsedLabel.SetText("已耗时 " + formatDuration(elapsed))
				})
			case <-ch:
				return
			}
		}
	}(stop)
}

func (ui *ConverterUI) stopTimer() {
	if ui.timerStop != nil {
		close(ui.timerStop)
		ui.timerStop = nil
	}
}

func (ui *ConverterUI) appendLog(message string) {
	fyne.Do(func() {
		ui.appendLogNow(message)
	})
}

func (ui *ConverterUI) appendLogNow(message string) {
	ui.logLines = append(ui.logLines, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message))
	if len(ui.logLines) > maxLogLines {
		ui.logLines = ui.logLines[len(ui.logLines)-maxLogLines:]
	}
	ui.logEntry.Enable()
	ui.logEntry.SetText(strings.Join(ui.logLines, "\n") + "\n")
	ui.logEntry.Disable()
}

func (ui *ConverterUI) finishWithError(err error) {
	fyne.Do(func() {
		ui.setRunningCommand(nil, false)
		ui.stopTimer()
		ui.progressInfinite.Stop()
		ui.progressInfinite.Hide()
		ui.progressBar.Show()
		ui.stopButton.Disable()
		if ui.ffmpegPath != "" {
			ui.startButton.Enable()
		}
		ui.statusLabel.SetText("状态：启动失败")
		ui.appendLogNow(err.Error())
		dialog.ShowError(err, ui.window)
	})
}

func (ui *ConverterUI) selectedPreset() Preset {
	category := ui.selectedCategory()
	label := ui.presetSelect.Selected
	for _, preset := range presets {
		if preset.Category == category && preset.Label == label {
			return preset
		}
	}
	for _, preset := range presets {
		if preset.Category == category {
			return preset
		}
	}
	return defaultPreset()
}

func (ui *ConverterUI) selectedCategory() string {
	if ui.categorySelect == nil || strings.TrimSpace(ui.categorySelect.Selected) == "" {
		return defaultPreset().Category
	}
	return ui.categorySelect.Selected
}

func defaultPreset() Preset {
	for _, preset := range presets {
		if preset.Key == defaultPresetKey {
			return preset
		}
	}
	return presets[0]
}

func categoryLabels() []string {
	seen := make(map[string]bool, len(presets))
	labels := make([]string, 0, len(presets))
	for _, preset := range presets {
		if seen[preset.Category] {
			continue
		}
		seen[preset.Category] = true
		labels = append(labels, preset.Category)
	}
	return labels
}

func presetLabelsForCategory(category string) []string {
	labels := make([]string, 0, len(presets))
	for _, preset := range presets {
		if preset.Category == category {
			labels = append(labels, preset.Label)
		}
	}
	return labels
}

func resolutionLabels() []string {
	labels := make([]string, 0, len(resolutionOptions))
	for _, option := range resolutionOptions {
		labels = append(labels, option.Label)
	}
	return labels
}

func frameModeLabels() []string {
	labels := make([]string, 0, len(frameModeOptions))
	for _, option := range frameModeOptions {
		labels = append(labels, option.Label)
	}
	return labels
}

func resolutionOptionByLabel(label string) ResolutionOption {
	for _, option := range resolutionOptions {
		if option.Label == label {
			return option
		}
	}
	return resolutionOptions[0]
}

func frameModeKeyByLabel(label string) string {
	for _, option := range frameModeOptions {
		if option.Label == label {
			return option.Key
		}
	}
	for _, option := range frameModeOptions {
		if option.Label == defaultFrameMode {
			return option.Key
		}
	}
	return ""
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func renderPresetArgs(preset Preset, crf int) []string {
	rendered := make([]string, 0, len(preset.Args))
	for _, token := range preset.Args {
		if token == "{crf}" {
			rendered = append(rendered, strconv.Itoa(crf))
			continue
		}
		rendered = append(rendered, token)
	}
	return rendered
}

func parsePositiveFloat(value string) float64 {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || number < 0 {
		return 0
	}
	return number
}

func parseFFmpegTimestamp(value string) float64 {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 3 {
		return 0
	}
	hours, err1 := strconv.Atoi(parts[0])
	minutes, err2 := strconv.Atoi(parts[1])
	seconds, err3 := strconv.ParseFloat(parts[2], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0
	}
	return float64(hours*3600+minutes*60) + seconds
}

func formatDuration(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds)
	hours := total / 3600
	minutes := (total % 3600) / 60
	secs := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func splitArgs(input string) ([]string, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}

	var (
		args    []string
		current strings.Builder
		quote   rune
		escaped bool
	)

	flush := func() {
		if current.Len() > 0 {
			args = append(args, current.String())
			current.Reset()
		}
	}

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if quote != 0 {
		return nil, errors.New("存在未闭合的引号")
	}
	if escaped {
		current.WriteRune('\\')
	}
	flush()
	return args, nil
}

func renderCommand(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" {
			quoted = append(quoted, `""`)
			continue
		}
		if strings.ContainsAny(arg, " \t\n\"'") {
			quoted = append(quoted, strconv.Quote(arg))
			continue
		}
		quoted = append(quoted, arg)
	}
	return strings.Join(quoted, " ")
}

func detectBinary(name string) string {
	candidates := make([]string, 0, 10)
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, name+ext),
			filepath.Join(exeDir, "bin", name+ext),
		)
		if runtime.GOOS == "darwin" {
			if resourcesDir := macAppResourcesDir(exeDir); resourcesDir != "" {
				candidates = append(candidates,
					filepath.Join(resourcesDir, name+ext),
					filepath.Join(resourcesDir, "bin", name+ext),
				)
			}
		}
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, name+ext),
			filepath.Join(wd, "bin", name+ext),
		)
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}

	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	if ext != "" {
		if path, err := exec.LookPath(name + ext); err == nil {
			return path
		}
	}
	return ""
}

func macAppResourcesDir(exeDir string) string {
	if filepath.Base(exeDir) != "MacOS" {
		return ""
	}
	contentsDir := filepath.Dir(exeDir)
	if filepath.Base(contentsDir) != "Contents" {
		return ""
	}
	return filepath.Join(contentsDir, "Resources")
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func safeCodec(codec string) string {
	if strings.TrimSpace(codec) == "" {
		return "unknown"
	}
	return codec
}

type runningSnapshot struct {
	cmd           *exec.Cmd
	stopRequested bool
	startedAt     time.Time
}

func (ui *ConverterUI) setRunningCommand(cmd *exec.Cmd, stopRequested bool) {
	ui.procMu.Lock()
	defer ui.procMu.Unlock()
	ui.currentCmd = cmd
	ui.stopRequested = stopRequested
	if cmd == nil {
		ui.startedAt = time.Time{}
	}
}

func (ui *ConverterUI) runningState() runningSnapshot {
	ui.procMu.Lock()
	defer ui.procMu.Unlock()
	return runningSnapshot{
		cmd:           ui.currentCmd,
		stopRequested: ui.stopRequested,
		startedAt:     ui.startedAt,
	}
}

func (ui *ConverterUI) isRunning() bool {
	ui.procMu.Lock()
	defer ui.procMu.Unlock()
	return ui.currentCmd != nil
}

func (ui *ConverterUI) ffmpegPathOrFallback() string {
	if ui.ffmpegPath != "" {
		return ui.ffmpegPath
	}
	return "ffmpeg"
}

func revealInFileManager(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	case "windows":
		return exec.Command("explorer", "/select,", path).Start()
	default:
		return exec.Command("xdg-open", filepath.Dir(path)).Start()
	}
}

type ioReadCloser interface {
	Read(p []byte) (n int, err error)
	Close() error
}
