package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var indexFilePath string

//var TickSpeedFactor int = 1
//var EventFrequencyFactor int = 1
//var Paused bool = false

func getSession(r *http.Request) string {
	cookie, err := r.Cookie("Session")
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return cookie.Value
}

func main() {

	initWeb()
	//InitLiftsSystem()
	r := mux.NewRouter()

	r.HandleFunc("/updates", serveWs)

	r.HandleFunc("/api/pause", func(w http.ResponseWriter, r *http.Request) {
		id := getSession(r)
		LiftSystems[id].State.Paused = true

	}).Methods("POST")

	r.HandleFunc("/api/unpause", func(w http.ResponseWriter, r *http.Request) {
		id := getSession(r)
		LiftSystems[id].State.Paused = false
	}).Methods("POST")

	r.HandleFunc("/api/speed", func(w http.ResponseWriter, r *http.Request) {
		id := getSession(r)
		r.ParseForm()
		values := r.PostForm["speed"]
		speed, _ := strconv.Atoi(values[0])
		LiftSystems[id].State.Speed = speed
	}).Methods("POST")

	r.HandleFunc("/api/eventfrequency", func(w http.ResponseWriter, r *http.Request) {
		id := getSession(r)
		r.ParseForm()
		values := r.PostForm["eventfrequency"]
		eventFrequency, _ := strconv.Atoi(values[0])
		LiftSystems[id].State.EventSpeed = eventFrequency
	}).Methods("POST")

	r.HandleFunc("/api/lift/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := getSession(r)
		vars := mux.Vars(r)
		liftId := vars["id"]

		lift := LiftSystems[id].GetLift(liftId)

		json.NewEncoder(w).Encode(lift)

	}).Methods("GET")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		id := getSession(r)
		if id != "" {
			_, present := LiftSystems[id]
			if present == false {
				id = ""
			}
		}
		if id == "" {
			newSession(w)
		}

		f, _ := os.Open(indexFilePath)
		io.Copy(w, f)
	}).Methods("GET")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("resources"))))

	http.ListenAndServe(":8080", r)
}

func initWeb() {

	var allFiles []string
	files, err := ioutil.ReadDir("./templates")
	if err != nil {
		fmt.Println(err)
	}
	for _, file := range files {
		filename := file.Name()
		if strings.HasSuffix(filename, ".tmpl") {
			allFiles = append(allFiles, "./templates/"+filename)
		}
	}

	indexFile, _ := ioutil.TempFile("", "liftsim_index")
	indexFilePath = indexFile.Name()

	var templates *template.Template
	templates, err = template.ParseFiles(allFiles...)

	templates.ExecuteTemplate(indexFile, "content", nil)
	indexFile.Close()

}

func newSession(w http.ResponseWriter) {

	sessionId := uuid.New()
	fmt.Printf("New Session %s\n", sessionId)
	cookie := new(http.Cookie)
	cookie.Name = "Session"
	cookie.Value = sessionId.String()
	http.SetCookie(w, cookie)

	liftSystem := NewLiftSystem(sessionId.String())
	go ticker(liftSystem)

}

var upgrader = websocket.Upgrader{}

func serveWs(w http.ResponseWriter, r *http.Request) {

	//	sessionId := uuid.New()
	id := getSession(r)

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("upgrade:", err)
		return
	}

	//	cookie := new(Cookie)
	//	cookie.Name = "Session"
	//	cookie.Value = sessionId.String()
	//	SetCookie(w, cookie)

	LiftSystems[id].Clients[ws] = true

}

func ticker(liftSystem *LiftSystemT) {

	for {
		ticker := time.NewTicker(time.Second / time.Duration(liftSystem.State.Speed))
		speed := liftSystem.State.Speed
		count := 0

		for now := range ticker.C {
			_ = now
			count++
			if liftSystem.State.Speed == 0 || liftSystem.State.Paused == true {
			} else if speed != liftSystem.State.Speed {
				ticker.Stop()
				break
			} else {
				go liftSystem.QueueTick(&Command{Tick, "", 0, NoDirection, nil})
			}

			freq := 11 - liftSystem.State.EventSpeed
			if count%freq == 0 {
				generateEvent(liftSystem)
				count = 0
			}
		}
	}
}

func generateEvent(liftSystem *LiftSystemT) {

	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	floor := random.Intn(49) + 1

	var b *Command

	destFloor := 1
	dir := Down
	if floor == 1 {
		destFloor = random.Intn(48) + 2
		dir = Up
	}

	p := NewPassengerForPickup(floor, destFloor, Up)

	b = &Command{PickupRequest, "", floor, dir, p}

	if b != nil {
		go liftSystem.QueueCommand(b)
	}
}
