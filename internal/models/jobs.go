package models

import "time"

type JobStatus string
type BidStatus string
type TransactionType string

const (
	JobOpen      JobStatus = "Open"
	JobAwarded   JobStatus = "Awarded"
	JobCompleted JobStatus = "Completed"
	JobCancelled JobStatus = "Cancelled"

	BidSubmitted BidStatus = "Submitted"
	BidAccepted  BidStatus = "Accepted"
	BidRejected  BidStatus = "Rejected"

	TransactionInitGrant    TransactionType = "InitialGrant"
	TransactionBidPlacement TransactionType = "BidPlacement"
	TransactionAdminAdj     TransactionType = "AdminAdjustment"
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

type Bid struct {
	ID             int64
	JobID          int64
	TradespersonID int64
	JobTitle       string
	Amount         string
	Message        string
	Status         BidStatus
	CreatedAt      time.Time
}

type TokenLedger struct {
	ID             int64
	TradespersonID int64
	ChangeAmount   string
	Transaction    TransactionType
	RelatedJobID   int64
	AdminUserId    int64
}

type Messages struct {
	ID             int64
	RelatedJobID   int64
	SenderUserID   int64
	ReceiverUserID int64
	Content        string
	IsRead         bool
}

type Ratings struct {
	ID             int64
	JobID          int64
	ReviewerUserID int64
	TradespersonID int64
	InferredScore  int64
	PublicComment  string
	PrivateComment string
	IsFlagged      bool
}

type ReviewQuestions struct {
	ID           int64
	QuestionText string
	IsActive     bool
}

type ReviewAnswers struct {
	ID         int64
	RatingID   int64
	QuestionID int64
	answer     string
}
