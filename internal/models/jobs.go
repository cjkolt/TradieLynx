package models

import "time"

type JobStatus string
type BidStatus string

const (
	JobOpen      JobStatus = "Open"
	JobAwarded   JobStatus = "Awarded"
	JobCompleted JobStatus = "Completed"
	JobCancelled JobStatus = "Cancelled"

	BidSubmitted BidStatus = "Submitted"
	BidAccepted  BidStatus = "Accepted"
	BidRejected  BidStatus = "Rejected"
)

type Job struct {
	ID                 int64
	HomeownerID        int64
	Title              string
	Description        string
	PostCode           string
	PreferredTimeframe string
	BudgetMinMax       string
	Status             JobStatus
}

type JobFilter struct {
	Query    string
	PostCode string
}

type Bid struct {
	ID               int64
	JobID            int64
	TradespersonID   int64
	TradespersonName string
	JobTitle         string
	Amount           string
	Message          string
	Status           BidStatus
	CreatedAt        time.Time
}
