package main

import (
	"log"
	"io/ioutil"
	"encoding/json"

	"bridge/sui"
)

func main() {
	config := sui.SuiEventConfig{ 
		EventHandle: "0x41c1082a0fdd7d4d333c67a882638dbc754a8d1e::memo_pool::Deposit",  
		Start: sui.EventID{
			TxSeq: 0, 
			EventSeq: 0, 
		}, 
		Limit: 10, 
	}

	data, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(data))

	ioutil.WriteFile("./config.json", data, 0644)
}