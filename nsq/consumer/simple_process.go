package consumer

import (
	"log"
	"encoding/json"
	
	"github.com/bitly/go-nsq"
)

func SimpleProcess(msg *nsq.Message) error {
	data := map[string]string{}
	if err := json.Unmarshal(msg.Body, &data); err != nil {
		log.Printf("NSQ Error : %s", err.Error())
		msg.Finish()
		return err
	}

	log.Printf("This is from NSQ: Message=%s&ID=%s", data["message"], data["id"])
	msg.Finish()
	return nil
}