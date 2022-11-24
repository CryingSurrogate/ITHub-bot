package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func cronProcess() {
	db := connectDb()
	defer db.Close()

	var processes map[int]NatsAd = make(map[int]NatsAd)

	for {
		// get db entries
		rows, err := db.Query("SELECT * FROM ads WHERE time - ? < 3600", time.Now().Unix())
		if err != nil {
			fmt.Println(err)
		}
		for rows.Next() {
			var id int
			var rowData NatsAd

			rows.Scan(&id, &rowData.Msg, &rowData.Time)

			if _, ok := processes[id]; !ok {
				go sendWithDelay(int(rowData.Time-time.Now().Unix()), rowData.Msg, id)
				processes[id] = rowData
			}
		}

		time.Sleep(10 * time.Minute)
	}
}

func sendWithDelay(delay int, msg string, id int) {
	time.Sleep(time.Duration(delay) * time.Second)
	// wait
	for _, v := range idcache {
		// send message
		txtmsg := SendMessage{
			ChId: v.Id,
			Text: msg,
		}
		bytemsg, _ := json.Marshal(txtmsg)
		_, err := http.Post(apiUrl+"/sendMessage", "application/json", bytes.NewReader(bytemsg))
		if err != nil {
			fmt.Println(err)
		}
	}
	// clean up db
	db := connectDb()
	defer db.Close()

	db.Exec("DELETE FROM ads WHERE id=?", id)
}
