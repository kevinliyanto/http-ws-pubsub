package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"

	"github.com/gorilla/mux"
)

var subscribersSlice []string
var subscribersMutex sync.RWMutex

type SubscribePayload struct {
	//topic string `json:"topic,omitempty"`
	url string `json:"url"`
}

type ErrorPayload struct {
	error string   `json:"error"`
	url   []string `json:"url, omitempty"`
}

func subscribe(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	payload := &SubscribePayload{}

	err = json.Unmarshal(reqBody, payload)
	if err != nil || !IsUrl(payload.url) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	subscribersMutex.Lock()
	defer subscribersMutex.Unlock()

	index := sort.SearchStrings(subscribersSlice, payload.url)
	if index == len(subscribersSlice) {
		subscribersSlice = append(subscribersSlice, payload.url)
		sort.Strings(subscribersSlice)
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusForbidden)
		e, _ := json.Marshal(&ErrorPayload{
			error: "URL is already registered",
		})
		_, _ = w.Write(e)
	}
}

func unsubscribe(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	payload := &SubscribePayload{}

	err = json.Unmarshal(reqBody, payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	subscribersMutex.Lock()
	defer subscribersMutex.Unlock()

	index := sort.SearchStrings(subscribersSlice, payload.url)
	if index == len(subscribersSlice) {
		w.WriteHeader(http.StatusNotFound)
		e, _ := json.Marshal(&ErrorPayload{
			error: "URL is not registered",
		})
		_, _ = w.Write(e)
	} else {
		subscribersSlice = append(subscribersSlice[:index], subscribersSlice[index+1:]...)
		w.WriteHeader(http.StatusOK)
	}
}

func publish(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	subscribersMutex.RLock()
	defer subscribersMutex.RUnlock()

	var failed []string
	for i := 0; i < len(subscribersSlice); i++ {
		_, err := http.Post(subscribersSlice[i], "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			failed = append(failed, subscribersSlice[i])
		}
	}

	if len(failed) > 0 {
		w.WriteHeader(http.StatusConflict)
		e, _ := json.Marshal(&ErrorPayload{
			error: "Cannot publish to URLs",
			url:   failed,
		})
		_, _ = w.Write(e)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func subscriber(w http.ResponseWriter, r *http.Request) {
	subscribersMutex.RLock()
	defer subscribersMutex.RUnlock()

	result, _ := json.Marshal(subscribersSlice)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result)
}

func IsUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func main() {
	// By default, this program listens on port 8080
	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	router := mux.NewRouter().StrictSlash(false)

	router.HandleFunc("/subscribe", subscribe).Methods("POST")
	router.HandleFunc("/unsubscribe", unsubscribe).Methods("POST")
	router.HandleFunc("/publish", publish).Methods("POST")
	router.HandleFunc("/subscriber", subscriber).Methods("GET")

	err := http.ListenAndServe(":"+port, router)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
}
