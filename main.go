package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Acrylic125/webhook-ingest-ws/ws"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
)

// TokenPairWebhookBody represents the top-level structure for a single
// TOKEN_PAIR_EVENT webhook.
type TokenPairWebhookBody struct {
	DeduplicationID string `json:"deduplicationId" validate:"required"`
	// GroupID         string               `json:"groupId" validate:"required"`
	Hash string `json:"hash" validate:"required"`
	// Type            string               `json:"type" validate:"required"`
	// Webhook         TokenPairWebhookInfo `json:"webhook" validate:"required"`
	// WebhookID       string               `json:"webhookId" validate:"required"`
	Data []TokenPairEventData `json:"data" validate:"required,dive"`
}

// TokenPairWebhookInfo holds information about the webhook itself.
// type TokenPairWebhookInfo struct {
// 	ID            string `json:"id" validate:"required"`
// 	Name          string `json:"name" validate:"required"`
// }

// TokenPairEventData holds the actual event and pair data for a token pair event.
type TokenPairEventData struct {
	Event TokenPairEvent `json:"event" validate:"required"`
	Pair  Pair           `json:"pair" validate:"required"`
}

// TokenPairEvent represents a specific token pair event.
type TokenPairEvent struct {
	Address string `json:"address" validate:"required"`
	// BaseTokenPrice     string                 `json:"baseTokenPrice" validate:"required"`
	// BlockHash          string                 `json:"blockHash" validate:"required"`
	// BlockNumber        int                    `json:"blockNumber" validate:"required"`
	Data             EventData `json:"data" validate:"required"`
	EventDisplayType string    `json:"eventDisplayType" validate:"required"`
	EventType        string    `json:"eventType" validate:"required"`
	EventType2       string    `json:"eventType2" validate:"required"`
	// ID                 string                 `json:"id" validate:"required"`
	// Labels             map[string]interface{} `json:"labels" validate:"required"` // Use interface{} for values if types vary
	LiquidityToken string `json:"liquidityToken" validate:"required"`
	// LogIndex           int                    `json:"logIndex" validate:"required"`
	Maker string `json:"maker" validate:"required"`
	// MakerHashKey       string                 `json:"makerHashKey" validate:"required"`
	// NetworkID          int                    `json:"networkId" validate:"required"`
	QuoteToken string `json:"quoteToken" validate:"required"`
	// SortKey            string                 `json:"sortKey" validate:"required"`
	// SupplementalIndex  int                    `json:"supplementalIndex" validate:"required"`
	Timestamp          int    `json:"timestamp" validate:"required"`
	Token0PoolValueUsd string `json:"token0PoolValueUsd" validate:"required"`
	Token0SwapValueUsd string `json:"token0SwapValueUsd" validate:"required"`
	Token0ValueBase    string `json:"token0ValueBase" validate:"required"`
	Token0ValueUsd     string `json:"token0ValueUsd" validate:"required"`
	Token1PoolValueUsd string `json:"token1PoolValueUsd" validate:"required"`
	Token1SwapValueUsd string `json:"token1SwapValueUsd" validate:"required"`
	Token1ValueBase    string `json:"token1ValueBase" validate:"required"`
	Token1ValueUsd     string `json:"token1ValueUsd" validate:"required"`
	// TransactionHash    string                 `json:"transactionHash" validate:"required"`
	// TransactionIndex   int                    `json:"transactionIndex" validate:"required"`
	// TTL                int                    `json:"ttl" validate:"required"`
}

// EventData represents the data specific to an event within a token pair event.
// This struct will need to be flexible as the 'data' field can vary significantly
// between different event types (e.g., "Swap" with different fields).
// I've included fields from both "Swap" examples you provided.
type EventData struct {
	Protocol string `json:"protocol" validate:"required"`
	Type     string `json:"type" validate:"required"` // e.g., "Swap"
}

// Pair represents the token pair information.
type Pair struct {
	Address      string `json:"address" validate:"required"`
	ExchangeHash string `json:"exchangeHash" validate:"required"`
	ID           string `json:"id" validate:"required"`
	NetworkID    int    `json:"networkId" validate:"required"`
	Token0       string `json:"token0" validate:"required"`
	Token1       string `json:"token1" validate:"required"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type HubManager struct {
	hub *ws.Hub[any]
}

func (h *HubManager) GetHub() *ws.Hub[any] {
	return h.hub
}

func (h *HubManager) OnRegister(client *ws.UserClient[any]) error {
	// fmt.Println("Client registered")
	return nil
}

func (h *HubManager) OnUnregister(client *ws.UserClient[any]) error {
	// fmt.Println("Client unregistered")
	return nil
}

func (h *HubManager) OnReceiveMessage(client *ws.UserClient[any], message []byte) error {
	// fmt.Println("Received message:", string(message))
	return nil
}

func main() {
	// hub := NewHub()
	hub := ws.NewHub[any]()
	hubManager := &HubManager{
		hub: hub,
	}
	go ws.Run(hubManager)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ConnectSocket(hubManager, w, r, nil)
	})
	http.HandleFunc("/send-data", func(w http.ResponseWriter, r *http.Request) {
		// Parse the incoming request body
		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// fmt.Println("Received data:", string(body))
		verify := TokenPairWebhookBody{}
		if err := json.Unmarshal(body, &verify); err != nil {
			fmt.Println("Error parsing JSON:", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		if err := validator.New().Struct(verify); err != nil {
			fmt.Println("Validation error:", err)
			http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		// SHA 256 hash "<secret><deduplicationId>"
		secret := "hello world"
		h := sha256.New()
		h.Write([]byte(secret + verify.DeduplicationID))
		hashInBytes := h.Sum(nil)
		hashString := hex.EncodeToString(hashInBytes)
		if hashString != verify.Hash {
			fmt.Println("Hash mismatch vvvvv - ", hashString, verify.Hash)
			http.Error(w, "Hash mismatch", http.StatusBadRequest)
			return
		}

		// Remarshal the data to ensure it is in the correct format
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(verify); err != nil {
			fmt.Println("Error encoding JSON:", err)
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			return
		}

		// fmt.Println("Parsed data:", verify)
		// Send the data to the WebSocket hub
		hub.Broadcast(buf.Bytes())

		// Send a success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Data received and broadcasted"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("WebSocket server starting on :8080")
	fmt.Println("Open http://localhost:8080 in your browser to test")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
