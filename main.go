package main

import (
	"log"
	"io/ioutil"
	"encoding/json"

	"bridge/aptos"
)

func main() {
	config := aptos.AptosEvnetConfig{
		Address: "0x8a51d48d19d02dec01fcf2014f5d04d21b97058346930ba21b85a61105c6b240", 
		EventHandle: "0x8a51d48d19d02dec01fcf2014f5d04d21b97058346930ba21b85a61105c6b240::memo_pool::MemoPool", 
		FieldName: "deposit_events", 
		Start: 0, 
		Limit: 10, 
	}

	data, err := json.Marshal(config)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(data))

	ioutil.WriteFile("./config.json", data, 0644)
}