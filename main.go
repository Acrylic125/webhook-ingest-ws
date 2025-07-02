package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Acrylic125/webhook-ingest-ws/ws"
	"github.com/gorilla/websocket"
)

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
	fmt.Println("Client registered")
	return nil
}

func (h *HubManager) OnUnregister(client *ws.UserClient[any]) error {
	fmt.Println("Client unregistered")
	return nil
}

func (h *HubManager) OnReceiveMessage(message []byte) error {
	fmt.Println("Received message:", string(message))
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

	fmt.Println("WebSocket server starting on :8080")
	fmt.Println("Open http://localhost:8080 in your browser to test")
	fmt.Println("Features:")
	fmt.Println("- Set keywords to filter incoming messages")
	fmt.Println("- Only receive messages containing your keywords")
	fmt.Println("- Leave keywords empty to receive all messages")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
