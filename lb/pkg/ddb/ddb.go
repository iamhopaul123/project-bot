package ddb

import (
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// ReviewerDB handles database actions to get reviewer info.
type ReviewerDB struct {
	svc       *dynamodb.DynamoDB
	tableName string
}

type Reviewer struct {
	Name    *string
	Point   *int64
	ChimeID *string
}

func NewReviewerDB(sess *session.Session, tableName string) *ReviewerDB {
	return &ReviewerDB{
		svc:       dynamodb.New(sess),
		tableName: tableName,
	}
}

func (db *ReviewerDB) Read() ([]Reviewer, error) {
	scanOut, err := db.svc.Scan(&dynamodb.ScanInput{
		TableName: &db.tableName,
	})
	if err != nil {
		return nil, err
	}
	var output []Reviewer
	for _, r := range scanOut.Items {
		var point *int64
		if r["point"] == nil {
			point = nil
		} else {
			p, err := strconv.ParseInt(*r["point"].N, 10, 64)
			if err != nil {
				return nil, err
			}
			point = &p
		}
		output = append(output, Reviewer{
			Name:    r["name"].S,
			Point:   point,
			ChimeID: r["chimeID"].S,
		})
	}
	return output, nil
}

func (db *ReviewerDB) Write(reviewers []Reviewer) error {
	for _, reviewer := range reviewers {
		_, err := db.svc.PutItem(&dynamodb.PutItemInput{
			TableName: &db.tableName,
			Item: map[string]*dynamodb.AttributeValue{
				"name": {
					S: reviewer.Name,
				},
				"point": {
					N: aws.String(strconv.FormatInt(*reviewer.Point, 10)),
				},
				"chimeID": {
					S: reviewer.ChimeID,
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
