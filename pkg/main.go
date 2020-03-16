package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/v29/github"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/oauth2"
)

const (
	PROJECT_NAME    = "Sprint"
	BACKLOG         = "Backlog"
	IN_PROGRESS     = "In progress"
	IN_REVIEW       = "In review"
	PENDING_RELEASE = "Pending release"
)

var (
	// GitHub owner name.
	owner = os.Getenv("GITHUB_OWNER")
	// GitHub repository name.
	repo = os.Getenv("GITHUB_REPO")
	// Private token of the GitHub Repo.
	secret = os.Getenv("GITHUB_TOKEN")
)

var allColumns = []string{BACKLOG, IN_PROGRESS, IN_REVIEW, PENDING_RELEASE}

func getColumns(ctx context.Context, client *github.Client, proj *github.Project) (map[string]*github.ProjectColumn, error) {
	projColumns := map[string]*github.ProjectColumn{
		BACKLOG:         nil,
		IN_PROGRESS:     nil,
		IN_REVIEW:       nil,
		PENDING_RELEASE: nil,
	}
	columns, _, err := client.Projects.ListProjectColumns(ctx, proj.GetID(), nil)
	if err != nil {
		return nil, err
	}
	for _, column := range columns {
		name := column.GetName()
		if _, ok := projColumns[name]; ok {
			projColumns[name] = column
		}
	}
	for k, v := range projColumns {
		if v == nil {
			return nil, fmt.Errorf("column %s does not exist", k)
		}
	}
	return projColumns, nil
}

func handler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// Validate payload.
	payload, err := github.ValidatePayload(req, []byte(os.Getenv("WEBHOOK_SECRET")))
	if err != nil {
		log.Printf("🚨 error validating request body: err=%s\n", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer req.Body.Close()

	// Parse payload to get the event.
	event, err := github.ParseWebHook(github.WebHookType(req), payload)
	if err != nil {
		log.Printf("🚨 error could not parse webhook: err=%s\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Auth to perform create/move card actions.
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: secret},
	)
	tc := oauth2.NewClient(ctx, ts)
	var client = github.NewClient(tc)

	switch e := event.(type) {
	case *github.PullRequestEvent:
		if e.GetAction() != "opened" {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		pr := e.GetPullRequest()

		// Get the project we want.
		projects, _, err := client.Repositories.ListProjects(ctx, owner, repo, nil)
		if err != nil {
			log.Printf("🚨 error getting project name: err=%s\n", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if projName := projects[0].GetName(); projName != PROJECT_NAME {
			log.Printf("🚨 error project %s not found: err=%s\n", projName, err)
			http.Error(w, fmt.Sprintf("project %s not found", projName), http.StatusNotFound)
			return
		}
		proj := projects[0]

		// Get the column info
		columns, err := getColumns(ctx, client, proj)
		if err != nil {
			log.Printf("🚨 error getting project columns: err=%s\n", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Get all cards in the project.
		var cards []*github.ProjectCard
		for _, columnName := range allColumns {
			columnCards, resp, err := client.Projects.ListProjectCards(ctx, columns[columnName].GetID(), nil)
			if err != nil {
				log.Printf("🚨 error listing project cards for column %s: err=%s\n", IN_REVIEW, err)
				http.Error(w, err.Error(), resp.StatusCode)
				return
			}
			cards = append(cards, columnCards...)
		}

		// Checkout if the card related to the PR already exists or not.
		cardID := int64(0)
		for _, card := range cards {
			if card.GetNodeID() == pr.GetNodeID() {
				cardID = card.GetID()
				break
			}
		}

		// If the card not exists, create a new card related to the PR in "In review" column.
		if cardID == 0 {
			_, resp, err := client.Projects.CreateProjectCard(ctx, columns[IN_REVIEW].GetID(), &github.ProjectCardOptions{
				ContentID:   pr.GetID(),
				ContentType: "PullRequest",
			})
			if err != nil {
				log.Printf("🚨 error creating project cards for pr %s: err=%s\n", pr.GetTitle(), err)
				http.Error(w, err.Error(), resp.StatusCode)
				return
			}
			w.WriteHeader(http.StatusCreated)
			log.Printf("✅ created a new pull-request project card in %s column\n", IN_REVIEW)
			return
		}
		// If exists, move the card to "In review" column.
		resp, err := client.Projects.MoveProjectCard(ctx, cardID, &github.ProjectCardMoveOptions{
			Position: "bottom",
			ColumnID: columns[IN_REVIEW].GetID(),
		})
		if err != nil {
			log.Printf("🚨 error moving project cards for pr %s: err=%s\n", pr.GetTitle(), err)
			http.Error(w, err.Error(), resp.StatusCode)
			return
		}
		w.WriteHeader(http.StatusCreated)
		log.Printf("✅ moved an existing pull-request project card to %s column\n", IN_REVIEW)
		return
	default:
		log.Printf("🤷‍♀️ event type %s\n", github.WebHookType(req))
		return
	}
}

func healthCheckHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("🚑 healthcheck ok!")
	w.WriteHeader(http.StatusOK)
}

func main() {

	router := httprouter.New()

	// Webhooks endpoint
	router.POST("/api/projectbot", handler)

	// Health Check
	router.GET("/", healthCheckHandler)

	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		header := w.Header()
		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Headers", "X-Requested-With")
		header.Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")

		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
	})

	log.Fatal(http.ListenAndServe(":80", router))
}
