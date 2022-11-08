package main

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattn/go-sqlite3"
)

const apiUrl = "https://api.telegram.org/" + "bot***"

func main() {
	sql.Register("sqlite3_with_extensions",
		&sqlite3.SQLiteDriver{
			Extensions: []string{
				"sqlite3_mod_regexp",
			},
		})

	go UpdateLoop()
	router := mux.NewRouter()
	router.HandleFunc("/api", IndexHandler)
	router.HandleFunc("/botName", NameHandler)
	router.HandleFunc("/eventId", EvIdHandler)
	router.HandleFunc("/lastId", LastIdHandler)
	router.HandleFunc("/login", Login)
	router.HandleFunc("/register", Register)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	http.ListenAndServe("localhost:8080", router)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	var M MainStruct

	resp, err := http.Get(apiUrl + "/getMe")
	if err != nil {
		fmt.Println(err)
	}
	respBody, _ := io.ReadAll(resp.Body)

	err = json.Unmarshal(respBody, &M) // заполнили перемнную m
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	M.Result.Abilities = append(M.Result.Abilities, "Replying to messages", "test")

	respReady, err := json.Marshal(M.Result)
	if err != nil {
		panic(err)
	}

	w.Write([]byte(respReady))
}

func NameHandler(w http.ResponseWriter, _ *http.Request) {
	db, err := sql.Open("sqlite3", "db.sql")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var gotname string
	var resp sql.NullString // для результата
	err = db.QueryRow("SELECT name FROM bot_status").Scan(&resp)
	if err != nil {
		fmt.Println(err)
	}
	if resp.Valid { // если результат валид
		gotname = resp.String // берём оттуда обычный string
	}
	w.Write([]byte(gotname))
}

func EvIdHandler(w http.ResponseWriter, _ *http.Request) {
	// db, err := sql.Open("sqlite3", "db.sql")
	// if err != nil {
	//     panic(err)
	// }
	// defer db.Close()
	// var goteventid string
	// var resp sql.NullString // для результата
	// err = db.QueryRow("SELECT id FROM bot_status").Scan(&resp)
	// if err != nil {
	//     fmt.Println(err)
	// }
	// if resp.Valid { // если результат валид
	//     goteventid = resp.String // берём оттуда обычный string
	// }
	// w.Write([]byte(goteventid))
}

func AuthCheck(w http.ResponseWriter, _ *http.Request) {

}

func Login(w http.ResponseWriter, r *http.Request) { // Тимонин Игоръ
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("BAD REQUEST"))
		return
	}
	var data UserLogin
	err = json.Unmarshal(reqBody, &data)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("BAD REQUEST"))
		return
	}
	// if data.Password != "" &&
	db, err := sql.Open("sqlite3", "file:db.sql")
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("INTERNAL DATABASE ERROR"))
		return
	}
	defer db.Close()
	var dbHashedPass string
	err = db.QueryRow("SELECT password FROM admins WHERE username = ?", data.Username).Scan(&dbHashedPass)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("USER NOT FOUND"))
		return
	}
	hash := md5.Sum([]byte(data.Password))
	hashString := hex.EncodeToString(hash[:])
	if dbHashedPass == hashString {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("SUCCESSFUL LOGIN"))
		return
	} else {
		fmt.Println(err)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("INCORRECT PASSWORD"))
		return
	}
}

func Register(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("BAD REQUEST"))
	}

	var data UserLogin
	err = json.Unmarshal(reqBody, &data)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("BAD REQUEST"))
	}

	db, err := sql.Open("sqlite3", "db.sql")
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("INTERNAL DATABASE ERROR"))
	}
	defer db.Close()

	rows, err := db.Query("SELECT id FROM admins WHERE username = ?", data.Username)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("INTERNAL DATABASE ERROR"))
	}

	if rows.Next() {
		fmt.Println("User already exists")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("USER EXISTS"))
	} else {

		hash := md5.Sum([]byte(data.Password))
		hashString := hex.EncodeToString(hash[:])

		_, err = db.Exec("INSERT INTO admins (username, password) VALUES (?, ?)", data.Username, hashString)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("INTERNAL DATABASE ERROR"))
		}
	}

}

func LastIdHandler(w http.ResponseWriter, _ *http.Request) {
	db, err := sql.Open("sqlite3", "db.sql")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var gotlastid string
	var resp sql.NullString // для результата
	err = db.QueryRow("SELECT lastid FROM bot_status").Scan(&resp)
	if err != nil {
		fmt.Println(err)
	}
	if resp.Valid { // если результат валид
		gotlastid = resp.String // берём оттуда обычный string
	}
	w.Write([]byte(gotlastid))
}

func UpdateLoop() {
	db, err := sql.Open("sqlite3", "db.sql")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	lastId := 0
	var nickname1 string
	err = db.QueryRow(`select name from bot_status`).Scan(&nickname1)
	if err != nil {
		fmt.Println(err)
	}
	for {
		newId := Update(lastId, &nickname1)
		if lastId != newId {
			lastId = newId
			db.Exec(`update bot_status set lastid = $1`, lastId)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func Update(lastId int, nickname *string) int {
	raw, err := http.Get(apiUrl + "/getUpdates?offset=" + strconv.Itoa(lastId))
	if err != nil {
		panic(err)
	}
	body, _ := io.ReadAll(raw.Body)

	var v UpdateResponse
	err = json.Unmarshal(body, &v)
	if err != nil {
		panic(err)
	}

	if len(v.Result) > 0 {
		ev := v.Result[len(v.Result)-1]
		txt := ev.Message.Text

		if strings.ToLower(txt) == "как тебя зовут?" {
			return WAY(lastId, ev, nickname)
		}
		if strings.Split(txt, ", ")[0] == *nickname {
			switch strings.Split(strings.Split(txt, ", ")[1], ": ")[0] {
			case "расскажи анекдот":
				{
					return Haha(lastId, ev)
				}
			case "дай предсказание на день":
				{
					return Predict(lastId, ev)
				}
			case "измени обращение на":
				{
					if strings.Contains(txt, ": ") {
						return ChangeName(lastId, ev, txt, nickname)
					} else {
						fmt.Println(err)
					}
				}
			default:
				{
					txtmsg := SendMessage{
						ChId:        ev.Message.Chat.Id,
						Text:        "Что нужно сделать?",
						Reply_to_id: ev.Message.Id,
					}
					bytemsg, _ := json.Marshal(txtmsg)
					_, err := http.Post(apiUrl+"/sendMessage", "application/json", bytes.NewReader(bytemsg))
					if err != nil {
						fmt.Println(err)
						return lastId
					} else {
						return ev.Id + 1
					}
				}
			}
		}
	}
	return lastId
}

func Predict(lastId int, ev UpdateStruct) int {
	txtmsg := SendMessage{
		ChId:        ev.Message.Chat.Id,
		Text:        "Сегодня чудный день для исполнения своих мечт",
		Reply_to_id: ev.Message.Id,
	}

	bytemsg, _ := json.Marshal(txtmsg)
	_, err := http.Post(apiUrl+"/sendMessage", "application/json", bytes.NewReader(bytemsg))
	if err != nil {
		fmt.Println(err)
		return lastId
	} else {
		return ev.Id + 1
	}
}

func ChangeName(lastId int, ev UpdateStruct, txt string, nickname *string) int {
	new := strings.Split(txt, "измени обращение на: ")
	*nickname = new[1]

	db, err := sql.Open("sqlite3", "db.sql")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	result, err := db.Exec(`update bot_status set name = $1`, *nickname)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.RowsAffected())

	txtmsg := SendMessage{
		ChId:        ev.Message.Chat.Id,
		Text:        "Теперь я " + *nickname,
		Reply_to_id: ev.Message.Id,
	}

	bytemsg, _ := json.Marshal(txtmsg)
	_, err = http.Post(apiUrl+"/sendMessage", "application/json", bytes.NewReader(bytemsg))
	if err != nil {
		fmt.Println(err)
		return lastId
	} else {
		return ev.Id + 1
	}
}

func WAY(lastId int, ev UpdateStruct, nickname *string) int {
	txtmsg := SendMessage{
		ChId:        ev.Message.Chat.Id,
		Text:        "Я " + *nickname,
		Reply_to_id: ev.Message.Id,
	}

	bytemsg, _ := json.Marshal(txtmsg)
	_, err := http.Post(apiUrl+"/sendMessage", "application/json", bytes.NewReader(bytemsg))
	if err != nil {
		fmt.Println(err)
		return lastId
	} else {
		return ev.Id + 1
	}
}
func Haha(lastId int, ev UpdateStruct) int {
	txtmsg := SendMessage{
		ChId:        ev.Message.Chat.Id,
		Text:        "Колобок повесился",
		Reply_to_id: ev.Message.Id,
	}

	bytemsg, _ := json.Marshal(txtmsg)
	_, err := http.Post(apiUrl+"/sendMessage", "application/json", bytes.NewReader(bytemsg))
	if err != nil {
		fmt.Println(err)
		return lastId
	} else {
		return ev.Id + 1
	}
}
