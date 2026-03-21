package wsclient

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"webrtc/rtc"

	"github.com/gorilla/websocket"
	pion "github.com/pion/webrtc/v3"
)

type WSClient struct {
	SendChan chan<- Event
	Close    func() error
}

// Event matches the structure used by the RTC manager
type Event struct {
	Type     string      `json:"type"`
	SenderID string      `json:"sender_id"`
	TargetID string      `json:"target_id"`
	Content  interface{} `json:"content"`
}

var (
	assignedID string
	idMu       sync.RWMutex
)

func StartWsClient(addr string, path string, savePath string, statusChan chan<- bool) (*WSClient, error) {
	u := url.URL{Scheme: "ws", Host: addr, Path: path}
	log.Printf("Client connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Dial error: %w", err)
	}

	// Setup PeerConnection
	pc, err := pion.NewPeerConnection(pion.Configuration{
		ICEServers: []pion.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("RTC error: %w", err)
	}

	manager := rtc.NewFileTransferManager(pc, conn)
	manager.PrepareReceiver(savePath)

	send := make(chan Event)

	wsc := &WSClient{
		SendChan: send,
		Close: func() error {
			pcErr := pc.Close()
			connErr := conn.Close()
			if connErr != nil {
				return connErr
			}
			return pcErr
		},
	}

	// Helper for decoding Content any -> WebRTC Structs
	decode := func(src any, dst any) {
		b, _ := json.Marshal(src)
		json.Unmarshal(b, dst)
	}

	// receiving Loop
	go func() {
		defer conn.Close()
		defer pc.Close()
		defer func() { statusChan <- false }()

		for {
			var event Event
			err := conn.ReadJSON(&event)
			if err != nil {
				return
			}

			switch event.Type {
			case "connected":
				statusChan <- true
				idMu.Lock()
				assignedID = fmt.Sprint(event.Content)
				idMu.Unlock()
				log.Printf("Client assigned ID: %s", assignedID)
				name, err := os.Hostname()
				if err != nil {
					fmt.Printf("Error retrieving device name: %v\n", err)
					name = fmt.Sprintf("%s_unkown", assignedID)
				}
				send <- Event{
					Type:     "identify",
					SenderID: assignedID,
					TargetID: "",
					Content:  name,
				}

			case "offer":
				log.Printf("Received offer from %s", event.SenderID)

				// Tell the manager who we are talking to for ICE candidates
				manager.SetTarget(event.SenderID)

				var offer pion.SessionDescription
				decode(event.Content, &offer)

				pc.SetRemoteDescription(offer)
				answer, _ := pc.CreateAnswer(nil)
				pc.SetLocalDescription(answer)

				send <- Event{
					Type:     "answer",
					TargetID: event.SenderID,
					Content:  answer,
				}

			case "answer":
				log.Printf("Received answer from %s", event.SenderID)
				var answer pion.SessionDescription
				decode(event.Content, &answer)
				pc.SetRemoteDescription(answer)

			case "candidate":
				var candidate pion.ICECandidateInit
				decode(event.Content, &candidate)
				pc.AddICECandidate(candidate)

			case "broadcast":
				log.Printf("[%s says]: %v", event.SenderID, event.Content)

			default:
				log.Printf("Client received unknown event: %s", event.Type)
			}
		}
	}()

	// Sending Loop
	go func() {
		defer conn.Close()
		for event := range send {
			event.SenderID = GetAssignedID()
			err := conn.WriteJSON(event)
			if err != nil {
				log.Printf("Client write error: %v", err)
				return
			}
		}
	}()

	return wsc, nil
}

func GetAssignedID() string {
	idMu.RLock()
	defer idMu.RUnlock()
	return assignedID
}
