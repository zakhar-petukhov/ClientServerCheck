package main

import (
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
	"sync"
)

var tmpl *template.Template // указатель на то где находится темплейтер

var client = New()                    // инициилизируем общую переменную для хранения подключений пользователей
var broadcast = make(chan MessageOut) // общий канал для рассылки сообщений всем клиента,рассылка осуществляется,чтобы

var upgrader = websocket.Upgrader{ReadBufferSize: 1024, // основные настройки для websocket соединения
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	}}

type Clients struct {
	Mutex       sync.RWMutex
	Connections map[*websocket.Conn]bool
}

// структура для ответа клиентам
type MessageOut struct {
	Counter int `json:"counter"`
}

// структура для получения комманд
type Message struct {
	Command string `json:"command"`
}

// метод инициализации новых клиентов на webscoket
func New() *Clients {
	return &Clients{
		Connections: make(map[*websocket.Conn]bool),
	}
}

// метод удаления клиентов из вебсокет с общим мютексом
func (f *Clients) Delete(ws *websocket.Conn) {
	f.Mutex.RLock()
	_, ok := f.Connections[ws]
	f.Mutex.RUnlock()
	if ok {
		f.Mutex.Lock()
		delete(f.Connections, ws)
		f.Mutex.Unlock()
	}
	// отправлем счетчик по событию удаление вебсокет соединения
	MessageOut := MessageOut{Counter: len(client.Connections)}
	broadcast <- MessageOut
	return
}

// метод удаление подключение без м.текса
func (f *Clients) delete(ws *websocket.Conn, e error) {
	if _, ok := f.Connections[ws]; ok {
		delete(f.Connections, ws)
	}
	MessageOut := MessageOut{Counter: len(client.Connections)}
	broadcast <- MessageOut
	return
}

// метод добавляем указатель на веб сокет соединение в словарь(map)
func (f *Clients) SetStatus(key *websocket.Conn, value bool) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	f.Connections[key] = true
	MessageOut := MessageOut{Counter: len(client.Connections)}
	broadcast <- MessageOut
}

// обработка всех подключений по websocket в бесконечном цикле до ошибки
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade http GET to websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()
	// добавляем ws структуру
	client.SetStatus(ws, true)
	for {
		var msg Message
		// декодируем сообщение из json
		err := ws.ReadJSON(&msg)
		if err != nil {
			client.Delete(ws)
			break
		}
		switch msg.Command {
		case "count":
			// на комманду count возвращает счетчик колл-ва подключенных пользователей
			MessageOut := MessageOut{Counter: len(client.Connections)}
			broadcast <- MessageOut
		}

	}
}

// функция отправки данных клиентам в websocet
func handleMessages() {
	for {
		select {
		case messageOut := <-broadcast:
			client.Mutex.Lock()
			for key, _ := range client.Connections {
				err := key.WriteJSON(messageOut)
				if err != nil {
					key.Close()
					client.delete(key, err) // если ошибка то удаляем вебсокет подключение,используем метод без мютекса
				}
			}
			client.Mutex.Unlock()
		}
	}
}

func index(reswt http.ResponseWriter, req *http.Request) {
	tmpl.ExecuteTemplate(reswt, "index.html", nil)

}

func main() {
	tmpl = template.Must(template.ParseFiles("./static/index.html"))
	http.HandleFunc("/", index)
	http.HandleFunc("/ws", handleConnections)
	go handleMessages() // отдельный поток для отправки сообщений
	log.Println("Http сервер запущен на порту: 4567")
	err := http.ListenAndServe(":4567", nil)
	if err != nil {
		log.Fatal("Ошибка старта сервера:", err)
	}
}
