package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/iamhopaul123/pr-reviewer-load-balancer/pkg/lb"
	"github.com/julienschmidt/httprouter"
)

var (
	table = os.Getenv("REVIEWER_NAME")
)

// HealthCheck just returns true if the service is up.
func HealthCheck(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	log.Println("🚑 healthcheck ok!")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

// GetReviewerHandler returns a reviewer with minimal point and add point to it.
func GetReviewerHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	pointStr := ps.ByName("point")
	author := ps.ByName("author")
	point, err := strconv.ParseInt(pointStr, 10, 64)
	if err != nil {
		if err != nil {
			log.Printf("🚨 error could not convert point %s to int64: err=%s\n", pointStr, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	reviewerLB, err := lb.NewReviewerLoadBalancer(author)
	if err != nil {
		log.Printf("🚨 error could not init the reviewer lb: err=%s\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reviewer, err := reviewerLB.GetReviewer(point)
	if err != nil {
		log.Printf("🚨 error could not get a reviewer to assign: err=%s\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("✅ Successfully get a reviewer to assign from the ddb table")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%s,%s", *reviewer.Name, *reviewer.ChimeID)))
}

func main() {
	router := httprouter.New()
	router.POST("/get-reviewer/:author/:point", GetReviewerHandler)

	// Health Check
	router.GET("/", HealthCheck)

	log.Fatal(http.ListenAndServe(":80", router))
}
