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
	//"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	newline        = []byte{'\n'}
	space          = []byte{' '}
	indexFilePaths map[string]string
	Simulations    map[string]*LiftSystem
)

func getCurrentSimulation(r *http.Request) *LiftSystem {

	cookie, err := r.Cookie("SimID")
	if err != nil {
		return nil
	}
	v, has := Simulations[cookie.Value]
	if has {
		return v
	}
	return nil
}

func getSimIdFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("SimID")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func main() {
	indexFilePaths = make(map[string]string)
	Simulations = make(map[string]*LiftSystem)

	r := mux.NewRouter()

	r.HandleFunc("/updates", serveWs)

	r.HandleFunc("/api/lift/pause", func(w http.ResponseWriter, r *http.Request) {
		simId := getSimIdFromCookie(r)
		LiftSystems[simId].State.Paused = true

	}).Methods("POST")

	r.HandleFunc("/api/unpause", func(w http.ResponseWriter, r *http.Request) {
		simId := getSimIdFromCookie(r)
		LiftSystems[simId].State.Paused = false
	}).Methods("POST")

	r.HandleFunc("/api/speed", func(w http.ResponseWriter, r *http.Request) {
		simId := getSimIdFromCookie(r)
		r.ParseForm()
		values := r.PostForm["speed"]
		speed, _ := strconv.Atoi(values[0])
		fmt.Printf("simId = %s\n", simId)
		LiftSystems[simId].State.Speed = speed
	}).Methods("POST")

	r.HandleFunc("/api/eventfrequency", func(w http.ResponseWriter, r *http.Request) {
		simId := getSimIdFromCookie(r)
		r.ParseForm()
		values := r.PostForm["eventfrequency"]
		eventFrequency, _ := strconv.Atoi(values[0])
		LiftSystems[simId].State.EventSpeed = eventFrequency
	}).Methods("POST")

	r.HandleFunc("/api/lift/{id}", func(w http.ResponseWriter, r *http.Request) {
		simId := getSimIdFromCookie(r)
		vars := mux.Vars(r)
		liftId, _ := vars["id"]
		lift := LiftSystems[simId].GetLift(liftId)

		json.NewEncoder(w).Encode(lift)

	}).Methods("GET")

	r.HandleFunc("/api/newsim", func(w http.ResponseWriter, r *http.Request) {

		simId := newSession(w)
		initWeb(simId)
		htmlPath, _ := indexFilePaths[simId]
		f, _ := os.Open(htmlPath)
		io.Copy(w, f)
	}).Methods("GET")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		simId := getSimIdFromCookie(r)
		htmlPath, has := indexFilePaths[simId]
		if simId == "" || !has {
			simId := newSession(w)
			initWeb(simId)
			htmlPath, has = indexFilePaths[simId]
			fmt.Printf("setting up new Simulation: %s\n", simId)
		}

		f, _ := os.Open(htmlPath)

		io.Copy(w, f)

	}).Methods("GET")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("resources"))))

	fmt.Printf("Listening")
	http.ListenAndServe(":8080", r)
}

func initWeb(simId string) string {

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

	indexFile, _ := ioutil.TempFile("", "liftsim_index"+simId)
	indexFilePath := indexFile.Name()
	indexFilePaths[simId] = indexFilePath

	var templates *template.Template
	templates, err = template.ParseFiles(allFiles...)

	templates.ExecuteTemplate(indexFile, "content", nil)

	indexFile.Close()
	return indexFilePath

}

func CreateSimulation() string {
	simId := uuid.New().String()
	fmt.Printf("New Simulation %s\n", simId)

	sim := NewLiftSystem(simId)

	Simulations[simId] = sim

	return simId
}

func newSession(w http.ResponseWriter) string {

	simId := CreateSimulation()
	setCookie(w, simId)

	return simId

}

func setCookie(w http.ResponseWriter, simId string) {
	cookie := new(http.Cookie)
	cookie.Name = "SimID"
	cookie.Value = simId
	http.SetCookie(w, cookie)

}

var upgrader = websocket.Upgrader{}

func serveWs(w http.ResponseWriter, r *http.Request) {

	simId := getSimIdFromCookie(r)
	if simId == "" {
		fmt.Printf("blank sim ID\n")
		return
	}

	sim := Simulations[simId]
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("upgrade:", err)
		return
	}

	fmt.Printf("convert to websocke simId = %s\n", simId)
	client := NewClient(sim, ws, simId)
	go client.ProcessCommands()

}
