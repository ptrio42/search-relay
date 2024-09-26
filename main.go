package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/fiatjaf/eventstore/elasticsearch"
	"github.com/fiatjaf/eventstore/lmdb"
	"github.com/fiatjaf/khatru"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type RequestPayload struct {
	Text string `json:"text"`
}

type ResponsePayload struct {
	Result interface{} `json:"result"`
}

func main() {
	godotenv.Load(".env")

	relay := khatru.NewRelay()

	relay.Info.Name = "Swarmstr Question-Only Search Relay"
	relay.Info.PubKey = "f1f9b0996d4ff1bf75e79e4cc8577c89eb633e68415c7faf74cf17a07bf80bd8"
	relay.Info.Icon = "https://image.nostr.build/191da0052d50ae4c937c9fc3361bb514523bbc37f41ae4c5ff5e1fc995bddd2d.png"

	db := lmdb.LMDBBackend{Path: "./db"}
	if err := db.Init(); err != nil {
		panic(err)
	}

	search := &elasticsearch.ElasticsearchStorage{URL: getEnv("ES_URL")}
	if err := search.Init(); err != nil {
		panic(err)
	}

	fmt.Println("Elasticsearch initialized successfully")

	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent, search.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, func(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
		if len(filter.Search) > 0 {
			return search.QueryEvents(ctx, filter)
		} else {
			filterNoSearch := filter
			filterNoSearch.Search = ""
			return db.QueryEvents(ctx, filterNoSearch)
		}
	})
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent, search.DeleteEvent)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.RejectEvent = append(relay.RejectEvent, func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {

		if len(event.Tags) > 4 {
			return true, "Too many tags. 4 max"
		}

		if !isQuestion(event.Content) {
			return true, "Not a question"
		}
		return false, ""
	})

	fmt.Println("running on :3337")

	http.ListenAndServe(":3337", relay)
}

func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Fatalf("Environment variable %s not set", key)
	}
	return value
}

func isQuestion(imageUrl string) bool {
	payload := RequestPayload{
		Text: imageUrl,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return false
	}

	resp, err := http.Post("http://@node:3006/classify-text", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return false
	}

	// Parse the response JSON
	var responsePayload ResponsePayload
	err = json.Unmarshal(body, &responsePayload)
	if err != nil {
		fmt.Printf("Error parsing JSON response: %v\n", err)
		return false
	}

	// Print the result from the Node.js worker
	fmt.Printf("Result from node.js: %+v\n", responsePayload.Result)
	return responsePayload.Result == true
}
