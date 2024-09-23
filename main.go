package main

import (
	"context"
	"fmt"
	"github.com/fiatjaf/eventstore/elasticsearch"
	"github.com/fiatjaf/eventstore/lmdb"
	"github.com/fiatjaf/khatru"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
	"log"
	"net/http"
	"os"
)

func main() {
	godotenv.Load(".env")

	relay := khatru.NewRelay()

	relay.Info.Name = "Swarmstr Search Relay"
	relay.Info.PubKey = "f1f9b0996d4ff1bf75e79e4cc8577c89eb633e68415c7faf74cf17a07bf80bd8"
	relay.Info.Icon = ""

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
