package main

import (
	"fmt"
	"github.com/fiatjaf/eventstore/elasticsearch"
	"github.com/fiatjaf/eventstore/lmdb"
	"github.com/fiatjaf/khatru"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

func main() {

	godotenv.Load(".env")

	relay := khatru.NewRelay()

	relay.Info.Name = "Swarmstr Search Relay"
	relay.Info.PubKey = "f1f9b0996d4ff1bf75e79e4cc8577c89eb633e68415c7faf74cf17a07bf80bd8"
	//relay.Info.Description = "A relay accepting only GM notes!"
	relay.Info.Icon = ""

	db := &lmdb.LMDBBackend{Path: "./db"}
	os.MkdirAll(db.Path, 0755)
	if err := db.Init(); err != nil {
		panic(err)
	}

	//search := bluge.BlugeBackend{Path: "./search", RawEventStore: db}
	//if err := search.Init(); err != nil {
	//	panic(err)
	//}

	search := elasticsearch.ElasticsearchStorage{URL: "http://elastic:" + getEnv("ELASTIC_PASSWORD") + "@es:9200", IndexName: "search"}

	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent, search.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents, search.QueryEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent, search.DeleteEvent)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)

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
