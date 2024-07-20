package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	//"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// Message structure
type Message struct {
    Type       string `json:"type"`
    SDP        string `json:"sdp,omitempty"`
    Candidate  string `json:"candidate,omitempty"`
    SelfParent string `json:"selfParent,omitempty"`
    OtherParent string `json:"otherParent,omitempty"`
    Event      *Event `json:"event,omitempty"`
    TargetNode string `json:"targetNode,omitempty"` // New target node field
}

// Event structure
type Event struct {
    Transactions [][]byte
    SelfParent   string
    OtherParent  string
    Creator      string
    Timestamp    time.Time
    Signature    string
    Hash         string
    RoundCreated int
    Famous       *bool
    Witness      bool
    LamportTime  int
}

// WebRTC configuration
var (
    webrtcConfig = webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {
                URLs: []string{"stun:stun.l.google.com:19302"},
            },
        },
    }
)

// Hashgraph structure
type Hashgraph struct {
    Events      map[string]*Event
    Rounds      map[int][]*Event
    privateKey  *ecdsa.PrivateKey
    publicKey   *ecdsa.PublicKey
    mutex       sync.RWMutex
}

// Create a new Hashgraph
func NewHashgraph(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) *Hashgraph {
    return &Hashgraph{
        Events:     make(map[string]*Event),
        Rounds:     make(map[int][]*Event),
        privateKey: privateKey,
        publicKey:  publicKey,
    }
}

// Add event
func (hg *Hashgraph) AddEvent(event *Event) error {
    hg.mutex.Lock()
    defer hg.mutex.Unlock()

    eventHash := hashEvent(event)
    event.Hash = eventHash

    if err := signEvent(event, hg.privateKey); err != nil {
        return err
    }

    hg.Events[event.Hash] = event
    hg.Rounds[event.RoundCreated] = append(hg.Rounds[event.RoundCreated], event)

    return nil
}

// Hash event
func hashEvent(event *Event) string {
    hash := sha256.New()
    hash.Write([]byte(event.Creator))
    hash.Write([]byte(event.SelfParent))
    hash.Write([]byte(event.OtherParent))
    hash.Write([]byte(event.Timestamp.String())) // Fix type conversion
    for _, tx := range event.Transactions {
        hash.Write(tx)
    }
    return hex.EncodeToString(hash.Sum(nil))
}

// Sign event
func signEvent(event *Event, privateKey *ecdsa.PrivateKey) error {
    hash := sha256.Sum256([]byte(event.Hash))
    r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
    if err != nil {
        return err
    }
    signature := append(r.Bytes(), s.Bytes()...)
    event.Signature = hex.EncodeToString(signature)
    return nil
}

// Verify event signature
func verifyEventSignature(event *Event, publicKey *ecdsa.PublicKey) bool {
    hash := sha256.Sum256([]byte(event.Hash))
    signature, err := hex.DecodeString(event.Signature)
    if err != nil {
        return false
    }
    r := big.NewInt(0).SetBytes(signature[:len(signature)/2])
    s := big.NewInt(0).SetBytes(signature[len(signature)/2:])
    return ecdsa.Verify(publicKey, hash[:], r, s)
}

// Get online nodes list
func getNodes() ([]string, error) {
    resp, err := http.Get("http://localhost:8080/nodes")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var nodes []string
    if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
        return nil, err
    }
    return nodes, nil
}

// Create a new WebRTC connection
func createPeerConnection() (*webrtc.PeerConnection, error) {
    peerConnection, err := webrtc.NewPeerConnection(webrtcConfig)
    if err != nil {
        return nil, err
    }

    // Set ICE candidate handler
    peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
        if c == nil {
            return
        }
        log.Printf("ICE Candidate: %s\n", c.ToJSON().Candidate)
    })

    // Set ICE connection state handler
    peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
        log.Printf("ICE Connection State: %s\n", state.String())
    })

    return peerConnection, nil
}

func main() {
    // WebSocket server address
    addr := "localhost:8080"

    // Connect to WebSocket server
    u := url.URL{Scheme: "ws", Host: addr, Path: "/signal"}
    log.Printf("Connecting to %s", u.String())

    c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatal("Failed to dial:", err)
    }
    defer c.Close()

    // Create WebRTC PeerConnection
    peerConnection, err := createPeerConnection()
    if err != nil {
        log.Fatal("Failed to create PeerConnection:", err)
    }

    // Generate ECDSA key pair
    privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        log.Fatal("Failed to generate ECDSA key:", err)
    }

    publicKey := &privateKey.PublicKey
    hashgraph := NewHashgraph(privateKey, publicKey)

    go func() {
        for {
            // Read message
            _, message, err := c.ReadMessage()
            if err != nil {
                log.Println("Failed to read message:", err)
                return
            }

            // Handle message
            var msg Message
            if err := json.Unmarshal(message, &msg); err != nil {
                log.Println("Failed to parse message:", err)
                return
            }

            switch msg.Type {
            case "offer":
                log.Println("Received offer")
                // Handle SDP exchange
                localSDP, err := peerConnection.CreateAnswer(nil)
                if err != nil {
                    log.Println("Failed to handle SDP exchange:", err)
                    return
                }

                if err := peerConnection.SetLocalDescription(localSDP); err != nil {
                    log.Println("Failed to set local SDP:", err)
                    return
                }

                answer := Message{
                    Type: "answer",
                    SDP:  localSDP.SDP,
                }
                if err := c.WriteJSON(answer); err != nil {
                    log.Println("Failed to send answer:", err)
                    return
                }

            case "candidate":
                log.Println("Received ICE candidate")
                // Add ICE candidate
                candidate := webrtc.ICECandidateInit{
                    Candidate: msg.Candidate,
                }
                if err := peerConnection.AddICECandidate(candidate); err != nil {
                    log.Println("Failed to add ICE candidate:", err)
                    return
                }

            case "event":
                log.Println("Received event")
                // Verify event signature
                if !verifyEventSignature(msg.Event, publicKey) {
                    log.Println("Event signature verification failed")
                    return
                }

                // Add event to local Hashgraph
                if err := hashgraph.AddEvent(msg.Event); err != nil {
                    log.Println("Failed to add event:", err)
                    return
                }
            }
        }
    }()

    // Send offer
    offer, err := peerConnection.CreateOffer(nil)
    if err != nil {
        log.Fatal("Failed to create offer:", err)
    }

    // Set local SDP
    if err := peerConnection.SetLocalDescription(offer); err != nil {
        log.Fatal("Failed to set local SDP:", err)
    }

    // Wait for ICE candidate gathering to complete
    <-webrtc.GatheringCompletePromise(peerConnection)

    // Send offer to signaling server
    offerMsg := Message{
        Type: "offer",
        SDP:  peerConnection.LocalDescription().SDP,
    }
    if err := c.WriteJSON(offerMsg); err != nil {
        log.Fatal("Failed to send offer:", err)
    }

    // Get online nodes list
    nodes, err := getNodes()
    if err != nil {
        log.Fatal("Failed to get online nodes list:", err)
    }
    log.Printf("Online nodes list: %v", nodes)

    // User create and send event logic
    go func() {
        scanner := bufio.NewScanner(os.Stdin)
        for {
            log.Print("Enter the message to send: ")
            if scanner.Scan() {
                text := scanner.Text()
                if text == "" {
                    continue
                }

                // Select a target node
                if len(nodes) == 0 {
                    log.Println("No other online nodes")
                    continue
                }
                log.Println("Please select a target node:")
                for i, node := range nodes {
                    log.Printf("%d: %s\n", i+1, node)
                }

                var targetNodeIndex int
                for {
                    log.Print("Enter target node number: ")
                    if scanner.Scan() {
                        input := scanner.Text()
                        index, err := strconv.Atoi(input)
                        if err == nil && index > 0 && index <= len(nodes) {
                            targetNodeIndex = index - 1
                            break
                        }
                        log.Println("Invalid input, please enter a valid node number")
                    }
                }
                targetNode := nodes[targetNodeIndex]

                // Create a new event
                event := &Event{
                    Transactions: [][]byte{[]byte(text)},
                    SelfParent:   "selfParentHash",
                    OtherParent:  "otherParentHash",
                    Creator:      "userID",
                    Timestamp:    time.Now(),
                }

                // Add event to local Hashgraph
                if err := hashgraph.AddEvent(event); err != nil {
                    log.Println("Failed to add event:", err)
                }

                // Send event to target node
                eventMsg := Message{
                    Type:      "event",
                    Event:     event,
                    TargetNode: targetNode,
                }
                if err := c.WriteJSON(eventMsg); err != nil {
                    log.Println("Failed to send event:", err)
                }
            }
        }
    }()

    // Wait for terminal input to keep the program running
    log.Println("Press Ctrl+C to exit")
    select {}
}
