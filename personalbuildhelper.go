package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	paramChan chan *request
	ffmChan   chan []byte
)

func init() {
	paramChan = make(chan *request, 20)
	ffmChan = make(chan []byte, 20)
}

type request struct {
	Result *Result `json:"result"`
}

type Result struct {
	ServiceType string
	MetaData    *MetaData   `json:"metadata"`
	Parameters  *Parameters `json:"parameters"`
}

type MetaData struct {
	IntentName string `json:"intentName"`
}

type Parameters struct {
	Status       int
	Code         string `json:"build-source-type"`
	BuildType    string `json:"build-type"`
	Action       string
	Location     string
	AlexaMessage string
}

type Fulfillment struct {
	Speech      string `json:"speech"`
	Data        string `json:"data"`
	DisplayText string `json:"displayText"`
}

func hello(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return
	}
	log.Printf("method: %v\n", r.Method)
	log.Printf("URL: %+v\n", r.URL)
	log.Printf("request body: %s\n", body)
	header := w.Header()
	header.Add("Content-type", "application/json")
	req := &request{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return
	}
	req.Result.ServiceType = "Google Home"
	paramChan <- req
	ffm := <-ffmChan
	log.Printf("received fullfillment: %s\n", ffm)
	/*respBody, err := json.Marshal(ffm)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return
	}*/
	w.Write(ffm)
}

func helloAlexa(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ALEXA ERROR: %v\n", err)
		return
	}
	log.Printf("ALEXA method: %v\n", r.Method)
	log.Printf("ALEXA URL: %+v\n", r.URL)
	req := &request{
		Result: &Result{
			ServiceType: "Alexa",
			Parameters: &Parameters{
				AlexaMessage: string(body),
			},
		},
	}

	paramChan <- req
	w.Write([]byte("This is heisenberg from your heroku cloud"))
	log.Printf("ALEXA request body: %s\n", body)
}

var mux map[string]func(http.ResponseWriter, *http.Request)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	server := http.Server{
		Addr:    ":" + port,
		Handler: &myHandler{},
	}

	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/gh"] = hello
	mux["/al"] = helloAlexa
	mux["/system"] = handleSystemClient
	server.ListenAndServe()
}

type myHandler struct{}

//comment
func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := mux[r.URL.String()]; ok {
		h(w, r)
		return
	}
	io.WriteString(w, "My server: "+r.URL.String())
}

func handleSystemClient(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("System Client ERROR: %v\n", err)
		return
	}
	w.Header().Add("Content-type", "application/json")
	timer := time.NewTimer(20 * time.Second)
	if r.Method == "POST" {
		/*ffm := &Fulfillment{}
		err = json.Unmarshal(body, &ffm)
		if err != nil {
			log.Printf("System Client ERROR: %v\n", err)
			return
		}
		*/
		log.Printf("Before sending bodyn")
		ffmChan <- body
		log.Printf("After sending bodyn")
		return
	}
	var req *request
	select {
	case req = <-paramChan:
	case <-timer.C:
		req = &request{
			Result: &Result{
				Parameters: &Parameters{
					Status: -1,
				},
			},
		}

	}
	data, err := json.Marshal(req)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return
	}
	log.Printf("response body: %s\n", data)
	w.Write(data)
}
