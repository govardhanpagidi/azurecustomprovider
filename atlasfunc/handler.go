package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/mongodb-forks/digest"
	"go.mongodb.org/atlas/mongodbatlas"
)

var (
	toolName        = "azCustomProvider"
	defaultLogLevel = "warning"
	userAgent       = fmt.Sprintf("%s/%s (%s;%s)", toolName, "0.0.1", runtime.GOOS, runtime.GOARCH)
	baseURL         = "https://cloud.mongodb.com"
	pubKey          = ""
	pvtKey          = ""
)

type Project struct {
	Name  string
	OrgID string
}

func main() {
	listenAddr := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		listenAddr = ":" + val
	}
	http.HandleFunc("/", handler)
	log.Printf("About to listen on %s. Go to https://127.0.0.1%s/", listenAddr, listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		id := r.URL.Query().Get("id")
		project, err := getProject(w, r, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			fmt.Fprint(w, "GET failed")
			return
		}
		data, _ := json.Marshal(&project)
		w.WriteHeader(200)
		w.Write(data)
	case "PUT", "POST":
		project, err := createProject(w, r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Something bad happened!"))
			return
		}
		data, _ := json.Marshal(&project)
		w.WriteHeader(200)
		w.Write([]byte("Success"))
		w.Write(data)
	case "DELETE":
		id := r.URL.Query().Get("id")
		err := deleteProject(w, r, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			fmt.Fprint(w, "DELETE failed")
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("Success"))
	}
}

func createProject(w http.ResponseWriter, r *http.Request) (*mongodbatlas.Project, error) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var proj Project
	json.Unmarshal(reqBody, &proj)

	client, err := NewMongoDBClient()
	if err != nil {
		return nil, err
	}
	defaultSettings := true
	project, _, err := client.Projects.Create(context.Background(), &mongodbatlas.Project{
		Name:                      proj.Name,
		OrgID:                     proj.OrgID,
		WithDefaultAlertsSettings: &defaultSettings,
	}, &mongodbatlas.CreateProjectOptions{ProjectOwnerID: ""})
	if err != nil {
		log.Printf("Create - error: %+v", err)
		return nil, err
	}
	return project, err
}

func deleteProject(w http.ResponseWriter, r *http.Request, id string) error {
	client, err := NewMongoDBClient()
	if err != nil {
		return err
	}
	_, err = client.Projects.Delete(context.Background(), id)
	if err != nil {
		log.Printf("DELETE - error: %+v", err)
		return err
	}
	return err
}

func getProject(w http.ResponseWriter, r *http.Request, id string) (*mongodbatlas.Project, error) {
	client, err := NewMongoDBClient()
	if err != nil {
		return nil, err
	}
	project, _, err := client.Projects.GetOneProject(context.Background(), id)
	if err != nil {
		log.Printf("GET - error: %+v", err)
		return nil, err
	}
	return project, err
}

func NewMongoDBClient() (*mongodbatlas.Client, error) {
	client, err := newHTTPClient()
	if err != nil {
		return nil, err
	}
	opts := []mongodbatlas.ClientOpt{mongodbatlas.SetUserAgent(userAgent)}
	opts = append(opts, mongodbatlas.SetBaseURL(baseURL))

	mongodbClient, err := mongodbatlas.New(client, opts...)
	if err != nil {
		return nil, err
	}

	return mongodbClient, nil
}

func newHTTPClient() (*http.Client, error) {
	if val, ok := os.LookupEnv("ATLAS_PUBLIC_KEY"); ok {
		pubKey = val
	}
	if val, ok := os.LookupEnv("ATLAS_PRIVATE_KEY"); ok {
		pvtKey = val
	}

	t := digest.NewTransport(pubKey, pvtKey)
	return t.Client()
}
