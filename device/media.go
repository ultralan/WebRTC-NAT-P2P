package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	mp4 "github.com/yapingcat/gomedia/go-mp4"
)

const resourceDir = "resources"

// MediaCommand 前端发来的媒体控制命令
type MediaCommand struct {
	MediaAction string  `json:"mediaAction"`
	File        string  `json:"file,omitempty"`
	Rate        float64 `json:"rate,omitempty"`
	Position    uint64  `json:"position,omitempty"` // 毫秒
}

// MediaResponse 返回给前端的媒体控制响应
type MediaResponse struct {
	MediaResponse bool        `json:"mediaResponse"`
	Status        string      `json:"status"`
	Message       string      `json:"message,omitempty"`
	Data          interface{} `json:"data,omitempty"`
}

// mediaState 推流共享状态，所有字段通过 mu 保护
type mediaState struct {
	mu            sync.Mutex
	paused        bool
	speed         float64
	seekMs        int64 // -1 表示不 seek
	stopCh        chan struct{}
	running       bool
	dataChannel   *webrtc.DataChannel
	videoTrack    *webrtc.TrackLocalStaticSample
	totalDuration uint64 // 毫秒
	currentTime   uint64 // 毫秒
}

func newMediaState() *mediaState {
	return &mediaState{
		speed:  1.0,
		seekMs: -1,
		stopCh: make(chan struct{}),
	}
}

func sendMediaResponse(d *webrtc.DataChannel, status, message string, data interface{}) {
	resp := MediaResponse{
		MediaResponse: true,
		Status:        status,
		Message:       message,
		Data:          data,
	}
	respJSON, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal media response: %s\n", err)
		return
	}
	d.SendText(string(respJSON))
}

func listMediaFiles() ([]string, error) {
	entries, err := os.ReadDir(resourceDir)
	if err != nil {
		return nil, fmt.Errorf("无法读取资源目录: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mp4") {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

func handleMediaCommand(d *webrtc.DataChannel, cmd *MediaCommand, ms *mediaState) {
	switch cmd.MediaAction {
	case "list":
		files, err := listMediaFiles()
		if err != nil {
			sendMediaResponse(d, "error", err.Error(), nil)
			return
		}
		sendMediaResponse(d, "list", "媒体文件列表", files)

	case "play":
		if cmd.File == "" {
			sendMediaResponse(d, "error", "未指定文件", nil)
			return
		}

		ms.mu.Lock()
		if ms.running {
			close(ms.stopCh)
			time.Sleep(100 * time.Millisecond)
		}
		ms.stopCh = make(chan struct{})
		ms.running = true
		ms.paused = false
		ms.speed = 1.0
		ms.seekMs = -1
		ms.dataChannel = d
		stopCh := ms.stopCh
		ms.mu.Unlock()

		filePath := filepath.Join(resourceDir, cmd.File)
		sendMediaResponse(d, "playing", "正在播放: "+cmd.File, nil)

		go func() {
			err := streamMP4ToTrack(filePath, ms, stopCh)
			ms.mu.Lock()
			ms.running = false
			ms.mu.Unlock()
			if err != nil {
				log.Printf("Stream error: %v\n", err)
				sendMediaResponse(d, "error", "播放错误: "+err.Error(), nil)
			} else {
				sendMediaResponse(d, "stopped", "播放结束", nil)
			}
		}()

	case "stop":
		ms.mu.Lock()
		if ms.running {
			close(ms.stopCh)
			ms.running = false
		}
		ms.mu.Unlock()
		sendMediaResponse(d, "stopped", "已停止播放", nil)

	case "pause":
		ms.mu.Lock()
		ms.paused = true
		ms.mu.Unlock()
		sendMediaResponse(d, "paused", "已暂停", nil)

	case "resume":
		ms.mu.Lock()
		ms.paused = false
		ms.mu.Unlock()
		sendMediaResponse(d, "resumed", "已恢复", nil)

	case "speed":
		rate := cmd.Rate
		if rate <= 0 {
			rate = 1.0
		}
		ms.mu.Lock()
		ms.speed = rate
		ms.mu.Unlock()
		sendMediaResponse(d, "speed", fmt.Sprintf("倍速: %.1fx", rate), nil)

	case "seek":
		ms.mu.Lock()
		ms.seekMs = int64(cmd.Position)
		ms.mu.Unlock()
		sendMediaResponse(d, "seeked", fmt.Sprintf("已跳转到 %.1fs", float64(cmd.Position)/1000), nil)
	}
}

func streamMP4ToTrack(filePath string, ms *mediaState, stopCh chan struct{}) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开文件: %v", err)
	}
	defer f.Close()

	demuxer := mp4.CreateMp4Demuxer(f)

	tracks, err := demuxer.ReadHead()
	if err != nil {
		return fmt.Errorf("解析 MP4 头失败: %v", err)
	}

	hasVideo := false
	for _, t := range tracks {
		if t.Cid == mp4.MP4_CODEC_H264 {
			hasVideo = true
			log.Printf("Found H.264 track: %dx%d, %d samples\n", t.Width, t.Height, t.SampleCount)
			break
		}
	}
	if !hasVideo {
		return fmt.Errorf("未找到 H.264 视频轨道")
	}

	var frames []videoFrame

	for {
		pkt, err := demuxer.ReadPacket()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取 MP4 包失败: %v", err)
		}
		if pkt.Cid == mp4.MP4_CODEC_H264 {
			frameCopy := make([]byte, len(pkt.Data))
			copy(frameCopy, pkt.Data)
			frames = append(frames, videoFrame{data: frameCopy, pts: pkt.Pts})
		}
	}

	if len(frames) == 0 {
		return fmt.Errorf("未读取到 H.264 视频帧")
	}

	totalDuration := frames[len(frames)-1].pts
	ms.mu.Lock()
	ms.totalDuration = totalDuration
	ms.mu.Unlock()

	log.Printf("Loaded %d video frames, total duration: %dms\n", len(frames), totalDuration)

	lastProgressTime := time.Now()

	for i := 0; i < len(frames); i++ {
		// 检查停止
		select {
		case <-stopCh:
			log.Println("Stream stopped by user")
			return nil
		default:
		}

		// 一次加锁，读取所有控制状态
		ms.mu.Lock()
		paused := ms.paused
		seekMs := ms.seekMs
		ms.seekMs = -1
		speed := ms.speed
		track := ms.videoTrack
		dc := ms.dataChannel
		ms.mu.Unlock()

		// 暂停等待
		for paused {
			select {
			case <-stopCh:
				return nil
			case <-time.After(50 * time.Millisecond):
			}
			ms.mu.Lock()
			paused = ms.paused
			ms.mu.Unlock()
		}

		// seek 跳转
		if seekMs >= 0 {
			i = findFrameByPts(frames, uint64(seekMs))
			if i >= len(frames) {
				break
			}
		}

		frame := frames[i]

		// 异步进度上报（不阻塞推帧）
		if time.Since(lastProgressTime) >= time.Second {
			lastProgressTime = time.Now()
			pts := frame.pts
			go sendProgressAsync(dc, pts, totalDuration, speed, paused)
		}

		// 计算帧间隔
		var duration time.Duration
		if i+1 < len(frames) {
			diff := frames[i+1].pts - frame.pts
			duration = time.Duration(diff) * time.Millisecond
		} else {
			duration = 33 * time.Millisecond
		}
		if duration <= 0 || duration > 200*time.Millisecond {
			duration = 33 * time.Millisecond
		}

		if track == nil {
			return fmt.Errorf("video track is nil")
		}

		err := track.WriteSample(media.Sample{
			Data:     frame.data,
			Duration: duration,
		})
		if err != nil {
			log.Printf("WriteSample error: %v\n", err)
		}

		// 按倍速调整 sleep
		time.Sleep(time.Duration(float64(duration) / speed))
	}

	return nil
}

type videoFrame struct {
	data []byte
	pts  uint64
}

func findFrameByPts(frames []videoFrame, targetMs uint64) int {
	for i, f := range frames {
		if f.pts >= targetMs {
			return i
		}
	}
	return len(frames) - 1
}

func sendProgressAsync(dc *webrtc.DataChannel, current, total uint64, speed float64, paused bool) {
	if dc == nil {
		return
	}

	data := map[string]interface{}{
		"current": current,
		"total":   total,
		"speed":   speed,
		"paused":  paused,
	}
	resp := MediaResponse{
		MediaResponse: true,
		Status:        "progress",
		Data:          data,
	}
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return
	}
	dc.SendText(string(respJSON))
}
