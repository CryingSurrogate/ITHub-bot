package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

func checkCache(id int) bool {

	if _, ok := idcache[id]; !ok {
		return false
	}

	return true
}

func initCache() {
	db := connectDb()
	defer db.Close()

	rows, err := db.Query("SELECT * FROM groups")
	if err != nil {
		fmt.Println(err)
	}

	for rows.Next() {
		var info ChatInfo

		rows.Scan(&info.Id, &info.Name, &info.Size, &info.Owner)
		idcache[info.Id] = info
	}
}

func addToCache(id int) {
	var temp ChatInfo = ChatInfo{
		Id:    id,
		Name:  "",
		Size:  0,
		Owner: 0,
	}

	raw, err := http.Get(apiUrl + "/getChatMemberCount?chat_id=" + strconv.Itoa(id))
	if err != nil {
		panic(err)
	}
	body, _ := io.ReadAll(raw.Body)
	count, _ := strconv.ParseInt(string(body), 10, 64) // size

	temp.Size = int(count)

	raw, err = http.Get(apiUrl + "/getChatAdministrators?chat_id=" + strconv.Itoa(id))
	if err != nil {
		panic(err)
	}
	body, _ = io.ReadAll(raw.Body)

	var resp ChatMembersResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		fmt.Println(err)
	}

	for _, v := range resp.Result {
		if v.Status == "creator" {
			if v.IsAnonymous {
				temp.Owner = 0
			} else {
				temp.Owner = v.User.Id
			}
		}
	}

	idcache[id] = temp

	db := connectDb()
	defer db.Close()

	db.Exec("INSERT INTO groups VALUES (?,?,?,?)", temp.Id, temp.Name, temp.Size, temp.Owner)

}
