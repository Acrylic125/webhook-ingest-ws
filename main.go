package main

import (
	"bytes"
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
	// DeduplicationID string               `json:"deduplicationId" validate:"required"`
	// GroupID         string               `json:"groupId" validate:"required"`
	// Hash            string               `json:"hash" validate:"required"`
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

// serveHome serves a simple HTML page for testing
func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Test with Keyword Filtering</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 800px; margin: 0 auto; }
        .section { margin-bottom: 20px; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .messages { height: 300px; overflow-y: auto; border: 1px solid #ccc; padding: 10px; background: #f9f9f9; }
        input, button { margin: 5px; padding: 8px; }
        .keyword-tag { display: inline-block; background: #007bff; color: white; padding: 2px 8px; margin: 2px; border-radius: 12px; font-size: 12px; }
        .keyword-tag .remove { cursor: pointer; margin-left: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>WebSocket Test with Keyword Filtering</h1>
        
        <div class="section">
            <h3>Set Keywords (comma-separated)</h3>
            <input type="text" id="keywordInput" placeholder="Enter keywords (e.g., hello, world, test)">
            <button onclick="setKeywords()">Set Keywords</button>
            <div id="currentKeywords"></div>
        </div>
        
        <div class="section">
            <h3>Chat</h3>
            <div id="messages" class="messages"></div>
            <input type="text" id="messageInput" placeholder="Type a message...">
            <button onclick="sendMessage()">Send</button>
        </div>
    </div>
    
    <script>
        const ws = new WebSocket('ws://localhost:8080/ws');
        const messages = document.getElementById('messages');
        const keywordInput = document.getElementById('keywordInput');
        const currentKeywords = document.getElementById('currentKeywords');
        
        ws.onmessage = function(event) {
            const div = document.createElement('div');
            
            // Try to parse as JSON first
            try {
                const data = JSON.parse(event.data);
                if (data.type === 'filter') {
                    div.textContent = 'System: ' + data.content;
                    div.style.color = 'green';
                } else {
                    div.textContent = 'Received: ' + data.content;
                }
            } catch (e) {
                // Plain text message
                div.textContent = 'Received: ' + event.data;
            }
            
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
        };
        
        function setKeywords() {
            const keywords = keywordInput.value.split(',').map(k => k.trim()).filter(k => k.length > 0);
            const message = {
                type: 'filter',
                keywords: keywords
            };
            ws.send(JSON.stringify(message));
            updateKeywordDisplay(keywords);
        }
        
        function updateKeywordDisplay(keywords) {
            currentKeywords.innerHTML = '';
            if (keywords.length === 0) {
                currentKeywords.innerHTML = '<em>No keywords set (receiving all messages)</em>';
                return;
            }
            
            currentKeywords.innerHTML = '<strong>Current keywords:</strong> ';
            keywords.forEach(keyword => {
                const tag = document.createElement('span');
                tag.className = 'keyword-tag';
                tag.textContent = keyword;
                currentKeywords.appendChild(tag);
            });
        }
        
        function sendMessage() {
            const input = document.getElementById('messageInput');
            const message = {
                type: 'chat',
                content: input.value
            };
            ws.send(JSON.stringify(message));
            
            const div = document.createElement('div');
            div.textContent = 'Sent: ' + input.value;
            div.style.color = 'blue';
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
            
            input.value = '';
        }
        
        document.getElementById('messageInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
        
        // Initialize with no keywords
        updateKeywordDisplay([]);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
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

	http.HandleFunc("/", serveHome)
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

		headers := r.Header
		fmt.Println("Headers:", headers)

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
	fmt.Println("Features:")
	fmt.Println("- Set keywords to filter incoming messages")
	fmt.Println("- Only receive messages containing your keywords")
	fmt.Println("- Leave keywords empty to receive all messages")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
