package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"hashgraphserver/server" // Updated import path
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Message structure
type Message struct {
    Type       string `json:"type"`
    SDP        string `json:"sdp,omitempty"`
    Candidate  string `json:"candidate,omitempty"`
    SelfParent string `json:"selfParent,omitempty"`
    OtherParent string `json:"otherParent,omitempty"`
    Event      *server.Event `json:"event,omitempty"`
    TargetNode string `json:"targetNode,omitempty"` // New target node field
}

// Upgrade HTTP connection to WebSocket connection
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

// Session manager structure
type SessionManager struct {
    sessions map[string]*websocket.Conn
    mutex    sync.Mutex
}

var sessionManager = SessionManager{
    sessions: make(map[string]*websocket.Conn),
}

// Register new node
func registerNode(conn *websocket.Conn) string {
    sessionManager.mutex.Lock()
    defer sessionManager.mutex.Unlock()
    id := uuid.New().String()
    sessionManager.sessions[id] = conn
    return id
}

// Unregister node
func unregisterNode(id string) {
    sessionManager.mutex.Lock()
    defer sessionManager.mutex.Unlock()
    delete(sessionManager.sessions, id)
}

// Get online nodes list
func getNodesHandler(w http.ResponseWriter, r *http.Request) {
    nodes := server.HashgraphManagerInstance.GetNodes()
    json.NewEncoder(w).Encode(nodes)
}

// WebSocket connection handler
func signalHandler(w http.ResponseWriter, r *http.Request) {
    // Upgrade HTTP connection to WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Failed to upgrade to WebSocket:", err)
        return
    }
    defer conn.Close()

    // Register node and get unique ID
    nodeID := registerNode(conn)
    // Register node to Hashgraph manager
    server.HashgraphManagerInstance.RegisterNode(nodeID)
    defer unregisterNode(nodeID)

    for {
        // Read message
        _, message, err := conn.ReadMessage()
        if err != nil {
            log.Println("Failed to read message:", err)
            break
        }

        // Parse message
        var msg Message
        err = json.Unmarshal(message, &msg)
        if err != nil {
            log.Println("Failed to parse message:", err)
            continue
        }

        switch msg.Type {
        case "offer":
            log.Println("Received offer")
            // Handle offer forwarding logic here
        case "answer":
            log.Println("Received answer")
            // Handle answer forwarding logic here
        case "candidate":
            log.Println("Received ICE candidate")
            // Handle ICE candidate forwarding logic here
        case "event":
            log.Println("Received event")
            // Handle event information and update Hashgraph
            transactions := [][]byte{} // Example transactions
            privateKey := &ecdsa.PrivateKey{} // Example private key

            err := server.HashgraphManagerInstance.AddEvent(nodeID, msg.SelfParent, msg.OtherParent, transactions, privateKey)
            if err != nil {
                log.Println("Failed to add event to Hashgraph:", err)
            }

            // If the target node is itself, handle the event directly
            if msg.TargetNode == nodeID {
                log.Println("Target node is itself, handling event directly")
                continue
            }

            // Forward event to target node
            if targetConn, ok := sessionManager.sessions[msg.TargetNode]; ok {
                if err := targetConn.WriteJSON(msg); err != nil {
                    log.Println("Failed to forward event:", err)
                }
            } else {
                log.Println("Target node does not exist or has disconnected")
            }
        }
    }
}

func main() {
    // Initialize MongoDB connection
    server.HashgraphManagerInstance.InitMongoDB("mongodb://localhost:27017", "hashgraphDB")
    defer server.HashgraphManagerInstance.CloseMongoDB()

    http.HandleFunc("/signal", signalHandler)
    http.HandleFunc("/nodes", getNodesHandler)
    log.Println("Signal server started, listening on port: 8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
