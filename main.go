/*
Overall considerations:

  - j variable on for loop should be 0 not 1, because I was getting on the output amount-1.
  - I think that you don't need to scan all petnames and adjectives at every request, you
    should do it at the api startup. This reduces the overall time for each request.
  - So, if you cannot load the petnames file at the startup the api should panic.
  - adj.txt has 1347 lines, the hardcoded line suggests 1345
  - petnames.txt has 5899 lines, the hardcoded line suggests 5800
  - capabilities has 17 items not 18
  - Instead of having this numbers hardcoded, they should be computed at runtime
  - Adding error handling on unmarshalling operation
  - The string slices should be initialized empty rather than using a empty string  ""
  - The capabilities slices were empty at the end of the processing

Nitpicks:

  - Instead of using append to insert items in a slice you could already begin with this
    values
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type uniCorn struct {
	Name         string
	Capabilities []string
}

type LIFOStore struct {
	items []uniCorn
	lock  sync.Mutex
}

func (s *LIFOStore) Push(item uniCorn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.items = append(s.items, item)
}

func (s *LIFOStore) Pop() interface{} {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.items) == 0 {
		return nil
	}

	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item
}

func (s *LIFOStore) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.items)
}

// Logger for debugging purposes
var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)

var UnicornStore = &LIFOStore{}
var UnicornsRequested = map[string]int{}
var AvailablePetnames = []string{}
var AvailableAdjectives = []string{}
var Capabilities = []string{
	"super strong",
	"fullfill wishes",
	"fighting capabilities",
	"fly",
	"swim",
	"sing",
	"run",
	"cry",
	"change color",
	"talk",
	"dance",
	"code",
	"design",
	"drive",
	"walk",
	"talk chinese",
	"lazy",
}

func main() {
	loadFile("petnames.txt", &AvailablePetnames)
	loadFile("adj.txt", &AvailableAdjectives)
	logger.Println("file loaded successfully")

	interval := 5
	ticker := time.Tick(time.Duration(interval) * time.Second)
	go func() {
		for {
			select {
			case <-ticker:
				unicorn := createUnicorn()
				UnicornStore.Push(unicorn)
				logger.Printf("unicorn created: %v\n", unicorn)
			}
		}
	}()

	http.HandleFunc("/api/get-unicorn", GetUnicorn)
	http.HandleFunc("/api/check-unicorn", CheckUnicorn)
	http.ListenAndServe(":8888", nil)
}

func loadFile(filename string, data *[]string) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	var scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		*data = append(*data, scanner.Text())
	}
}

func createUnicorn() uniCorn {
	sleep_time := time.Duration(rand.Intn(1000)) * time.Millisecond
	chosenAdjective := AvailableAdjectives[rand.Intn(len(AvailableAdjectives))]
	chosenName := AvailablePetnames[rand.Intn(len(AvailableAdjectives))]
	time.Sleep(sleep_time)

	caps := make(map[string]interface{})
	for len(caps) < 3 {
		cap := Capabilities[rand.Intn(len(Capabilities))]
		caps[cap] = nil
	}

	uniqueCaps := []string{}
	for cap := range caps {
		uniqueCaps = append(uniqueCaps, cap)
	}

	return uniCorn{
		Name:         chosenAdjective + "-" + chosenName,
		Capabilities: uniqueCaps,
	}
}

func GetUnicorn(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	amount, err := strconv.Atoi(values.Get("amount"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	requestID, err := exec.Command("uuidgen").Output()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	strRequestID := strings.TrimRight(string(requestID), "\n")
	UnicornsRequested[strRequestID] = amount
	log.Printf("unicorns requested: %v", UnicornRequested)

	data := struct {
		Token  string
		Amount int
	}{
		Token:  strRequestID,
		Amount: amount,
	}

	d, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(d)
}

func CheckUnicorn(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	requestedAmount, exists := UnicornsRequested[token]
	if !exists {
		http.Error(w, "Token not found", http.StatusNotFound)
		return
	}

	if UnicornStore.Len() < requestedAmount {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Processing in progress..."))
	}

	items := []uniCorn{}
	for i := 0; i < requestedAmount; i++ {
		item, ok := UnicornStore.Pop().(uniCorn)
		if !ok {
			http.Error(w, "Could not procceed", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	d, err := json.Marshal(items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("unicorns ready")
	log.Println(items)
	w.Write(d)
}
