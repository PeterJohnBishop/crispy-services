package rtc

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type Event struct {
	Type     string      `json:"type"`
	SenderID string      `json:"sender_id"`
	TargetID string      `json:"target_id"`
	Content  interface{} `json:"content"`
}

type FileTransferManager struct {
	PC       *webrtc.PeerConnection
	WS       *websocket.Conn
	mu       sync.Mutex
	targetID string
}

func NewFileTransferManager(pc *webrtc.PeerConnection, ws *websocket.Conn) *FileTransferManager {
	m := &FileTransferManager{
		PC: pc,
		WS: ws,
	}

	m.PC.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		m.mu.Lock()
		target := m.targetID
		m.mu.Unlock()

		if target != "" {
			m.WS.WriteJSON(Event{
				Type:     "candidate",
				TargetID: target,
				Content:  i.ToJSON(),
			})
		}
	})

	return m
}

func (m *FileTransferManager) StartSender(targetID string, filePath string) error {
	m.mu.Lock()
	m.targetID = targetID
	m.mu.Unlock()

	err := m.streamFileThrottled(filePath)
	if err != nil {
		return fmt.Errorf("failed to setup data channel: %w", err)
	}

	offer, err := m.PC.CreateOffer(nil)
	if err != nil {
		return err
	}

	if err = m.PC.SetLocalDescription(offer); err != nil {
		return err
	}

	return m.WS.WriteJSON(Event{
		Type:     "offer",
		TargetID: targetID,
		Content:  offer,
	})
}

func (m *FileTransferManager) PrepareReceiver(savePath string) {
	m.PC.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("Data channel '%s' opened, receiving file...", dc.Label())

		file, err := os.Create(savePath)
		if err != nil {
			log.Printf("File creation error: %v", err)
			return
		}

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			// Check for string "EOF"
			if msg.IsString && string(msg.Data) == "EOF" {
				log.Println("File transfer complete.")
				file.Sync()
				file.Close()
				return
			}

			// Write incoming bytes
			if _, err := file.Write(msg.Data); err != nil {
				log.Printf("Write error: %v", err)
			}
		})
	})
}

func (m *FileTransferManager) SetTarget(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.targetID = id
}

func (m *FileTransferManager) streamFileThrottled(filePath string) error {
	dc, err := m.PC.CreateDataChannel("file-transfer", nil)
	if err != nil {
		return err
	}

	dc.SetBufferedAmountLowThreshold(512 * 1024) // 512 KB
	resume := make(chan struct{}, 1)

	dc.OnBufferedAmountLow(func() {
		select {
		case resume <- struct{}{}:
		default:
		}
	})

	dc.OnOpen(func() {
		log.Println("Data channel open, starting stream...")
		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("Open file error: %v", err)
			return
		}
		defer file.Close()

		buffer := make([]byte, 16384) // 16KB chunks
		for {
			if dc.BufferedAmount() > dc.BufferedAmountLowThreshold() {
				<-resume
			}

			n, err := file.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				return
			}

			if sendErr := dc.Send(buffer[:n]); sendErr != nil {
				return
			}
		}
		dc.SendText("EOF")
	})

	return nil
}
