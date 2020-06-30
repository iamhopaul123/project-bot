package lb

import (
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/bradfitz/slice"
	"github.com/iamhopaul123/pr-reviewer-load-balancer/pkg/ddb"
)

const (
	maximumPointForReviewerWithMinimumPoint = 1000
)

var (
	table = os.Getenv("REVIEWER_NAME")
)

type ReviewerLoadBalancer struct {
	author   string
	reviewer []ddb.Reviewer
	dbSvc    *ddb.ReviewerDB
}

func NewReviewerLoadBalancer(author string) (*ReviewerLoadBalancer, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	db := ddb.NewReviewerDB(sess, table)
	reviews, err := db.Read()
	if err != nil {
		return nil, err
	}
	return &ReviewerLoadBalancer{
		author:   author,
		reviewer: reviews,
		dbSvc:    db,
	}, nil
}

func (lb *ReviewerLoadBalancer) GetReviewer(point int64) (*ddb.Reviewer, error) {
	reviewers, err := lb.dbSvc.Read()
	if err != nil {
		return nil, err
	}
	if len(reviewers) == 0 {
		return nil, errors.New("no reviewers in the ddb table")
	}
	slice.Sort(reviewers[:], func(i, j int) bool {
		return *reviewers[i].Point < *reviewers[j].Point
	})
	var reviewer *ddb.Reviewer
	if *reviewers[0].Name != lb.author {
		reviewer = &ddb.Reviewer{
			Name:    reviewers[0].Name,
			Point:   reviewers[0].Point,
			ChimeID: reviewers[0].ChimeID,
		}
		*reviewers[0].Point += point
	} else {
		reviewer = &ddb.Reviewer{
			Name:    reviewers[1].Name,
			Point:   reviewers[1].Point,
			ChimeID: reviewers[1].ChimeID,
		}
		*reviewers[1].Point += point
	}
	slice.Sort(reviewers[:], func(i, j int) bool {
		return *reviewers[i].Point < *reviewers[j].Point
	})
	if *reviewers[0].Point >= maximumPointForReviewerWithMinimumPoint {
		for _, reviewer := range reviewers {
			*reviewer.Point -= maximumPointForReviewerWithMinimumPoint
		}
	}
	err = lb.dbSvc.Write(reviewers)
	if err != nil {
		return nil, err
	}
	return reviewer, nil
}

// foo,bar -> [foo, bar]
func reviewerParser(s string) []string {
	res := strings.Split(s, ",")
	return res
}
