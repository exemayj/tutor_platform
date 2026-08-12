package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutor_platform/internal/middleware"
)

type ChatHandler struct {
	DB       *pgxpool.Pool
	upgrader websocket.Upgrader
	clients  map[string]map[*websocket.Conn]*ClientInfo
	mu       sync.Mutex
}

type ClientInfo struct {
	UserID string
	Role   string
	Name   string
}

type ChatMessage struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	SenderID   string `json:"sender_id,omitempty"`
	SenderName string `json:"sender_name,omitempty"`
	Time       string `json:"time"`
}

func NewChatHandler(db *pgxpool.Pool) *ChatHandler {
	return &ChatHandler{
		DB: db,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "http://localhost:8080" || origin == "https://твой-домен.ru"
			},
		},
		clients: make(map[string]map[*websocket.Conn]*ClientInfo),
	}
}

func (h *ChatHandler) ChatPage(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	orderID := chi.URLParam(r, "orderID")

	var studentID, tutorUserID string
	var orderStatus string
	err := h.DB.QueryRow(r.Context(),
		`SELECT o.student_id, tp.user_id, o.status
		 FROM orders o
		 JOIN tutor_profiles tp ON o.tutor_profile_id = tp.id
		 WHERE o.id = $1`,
		orderID,
	).Scan(&studentID, &tutorUserID, &orderStatus)

	if err != nil {
		http.Error(w, "Заявка не найдена", http.StatusNotFound)
		return
	}

	if claims.UserID != studentID && claims.UserID != tutorUserID {
		http.Error(w, "Нет доступа", http.StatusForbidden)
		return
	}

	if orderStatus != "accepted" {
		http.Error(w, "Чат доступен только для принятых заявок", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Чат</title>
    <style>
        body { font-family: Arial; max-width: 600px; margin: 0 auto; padding: 20px; }
        #messages { border: 1px solid #ccc; height: 400px; overflow-y: scroll; padding: 10px; margin-bottom: 10px; }
        .msg { margin-bottom: 8px; }
        .time { color: #999; font-size: 12px; }
        .mine { text-align: right; }
        .mine .text { background: #007aff; color: white; padding: 8px 12px; border-radius: 12px; display: inline-block; }
        .other .text { background: #e9e9eb; padding: 8px 12px; border-radius: 12px; display: inline-block; }
        .system { text-align: center; color: #999; font-size: 12px; margin: 10px 0; }
        #input { display: flex; gap: 10px; }
        #input input { flex: 1; padding: 10px; }
        #input button { padding: 10px 20px; }
    </style>
</head>
<body>
    <h2>Чат</h2>
    <a href="/orders">← К заявкам</a>
    <div id="messages"></div>
    <div id="input">
        <input type="text" id="msgInput" placeholder="Сообщение...">
        <button id="sendBtn">Отправить</button>
    </div>

    <script>
        var currentUserID = "` + claims.UserID + `";
        var orderID = "` + orderID + `";

        console.log("Connecting to WebSocket...");
        console.log("currentUserID:", currentUserID);
        console.log("orderID:", orderID);

        var ws = new WebSocket("ws://localhost:8080/ws/chat/" + orderID);

        ws.onopen = function() {
            console.log("WebSocket connected!");
            var div = document.getElementById('messages');
            var msgDiv = document.createElement('div');
            msgDiv.className = 'system';
            msgDiv.textContent = 'Вы в чате';
            div.appendChild(msgDiv);
        };

        ws.onmessage = function(event) {
            console.log("Message received:", event.data);
            var msg = JSON.parse(event.data);
            var div = document.getElementById('messages');
            var msgDiv = document.createElement('div');

            if (msg.type === 'system') {
                msgDiv.className = 'system';
                msgDiv.textContent = msg.message;
            } else {
                msgDiv.className = 'msg ' + (msg.sender_id === currentUserID ? 'mine' : 'other');
                msgDiv.innerHTML = '<div><strong>' + msg.sender_name + '</strong> <span class="time">' + msg.time + '</span></div>' +
                                   '<div class="text">' + msg.message + '</div>';
            }
            div.appendChild(msgDiv);
            div.scrollTop = div.scrollHeight;
        };

        ws.onerror = function(err) {
            console.log("WebSocket error:", err);
        };

        ws.onclose = function() {
            console.log("WebSocket closed");
            var div = document.getElementById('messages');
            var msgDiv = document.createElement('div');
            msgDiv.className = 'system';
            msgDiv.textContent = 'Соединение закрыто';
            div.appendChild(msgDiv);
        };

        function sendMsg() {
            var input = document.getElementById('msgInput');
            var text = input.value.trim();
            if (!text) return;
            console.log("Sending:", text);
            ws.send(JSON.stringify({message: text}));
            input.value = '';
        }

        document.getElementById('sendBtn').onclick = sendMsg;

        document.getElementById('msgInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                sendMsg();
            }
        });
    </script>
</body>
</html>`))
}

func (h *ChatHandler) ChatWebSocket(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())

	orderID := chi.URLParam(r, "orderID")

	var studentID, tutorUserID string
	var orderStatus string
	err := h.DB.QueryRow(r.Context(),
		`SELECT o.student_id, tp.user_id, o.status
		 FROM orders o
		 JOIN tutor_profiles tp ON o.tutor_profile_id = tp.id
		 WHERE o.id = $1`,
		orderID,
	).Scan(&studentID, &tutorUserID, &orderStatus)

	if err != nil {
		http.Error(w, "Заявка не найдена", http.StatusNotFound)
		return
	}

	if claims.UserID != studentID && claims.UserID != tutorUserID {
		http.Error(w, "Нет доступа", http.StatusForbidden)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Регистрируем клиента
	h.mu.Lock()
	if h.clients[orderID] == nil {
		h.clients[orderID] = make(map[*websocket.Conn]*ClientInfo)
	}
	h.clients[orderID][conn] = &ClientInfo{
		UserID: claims.UserID,
		Role:   claims.Role,
		Name:   claims.Email,
	}
	h.mu.Unlock()

	// Оповещаем о входе (broadcast сам управляет мьютексом)
	h.broadcast(orderID, ChatMessage{
		Type:    "system",
		Message: claims.Email + " в чате",
		Time:    time.Now().Format("15:04"),
	}, conn)

	// Убираем клиента при выходе
	defer func() {
		h.mu.Lock()
		delete(h.clients[orderID], conn)
		if len(h.clients[orderID]) == 0 {
			delete(h.clients, orderID)
		}
		h.mu.Unlock()
	}()

	// Цикл чтения сообщений
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Клиент отключился: %v", err)
			break
		}

		var msg struct {
			Message string `json:"message"`
		}
		if len(msgBytes) > 4096 {
			continue
		}
		if err := json.Unmarshal(msgBytes, &msg); err != nil || msg.Message == "" {
			continue
		}

		now := time.Now()

		h.DB.Exec(r.Context(),
			"INSERT INTO messages (order_id, sender_id, message, created_at) VALUES ($1, $2, $3, $4)",
			orderID, claims.UserID, msg.Message, now,
		)

		h.broadcast(orderID, ChatMessage{
			Type:       "message",
			Message:    msg.Message,
			SenderID:   claims.UserID,
			SenderName: claims.Email,
			Time:       now.Format("15:04"),
		}, nil)
	}
}

func (h *ChatHandler) broadcast(orderID string, msg ChatMessage, except *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	msgJSON, _ := json.Marshal(msg)

	for conn := range h.clients[orderID] {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := conn.WriteMessage(websocket.TextMessage, msgJSON)
		if err != nil {
			conn.Close()
			delete(h.clients[orderID], conn)
		}
	}
}
