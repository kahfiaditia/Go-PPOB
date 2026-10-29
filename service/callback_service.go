package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"pulsa-backend/repository"
	"pulsa-backend/util"
)

type CallbackPayload struct {
	TxID       string    `json:"tx_id"`
	Status     string    `json:"status"`
	CustomerNo string    `json:"customer_no,omitempty"`
	ProductID  string    `json:"product_id,omitempty"`
	Amount     float64   `json:"amount,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type CallbackService struct {
	db         *sql.DB
	agentRepo  *repository.AgentRepository
	txRepo     *repository.TransactionRepository
	httpClient *http.Client
	maxRetry   int
}

func NewCallbackService(db *sql.DB, ar *repository.AgentRepository, tr *repository.TransactionRepository,
	timeoutSeconds, maxRetry int) *CallbackService {
	return &CallbackService{
		db:         db,
		agentRepo:  ar,
		txRepo:     tr,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
		maxRetry:   maxRetry,
	}
}

// Send — kirim callback async dengan retry & log
func (s *CallbackService) Send(agentID int64, payload CallbackPayload) {
	go s.sendWithRetry(agentID, payload)
}

func (s *CallbackService) sendWithRetry(agentID int64, payload CallbackPayload) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil || agent.CallbackURL == "" {
		util.Log.Debugf("[CALLBACK] skip tx=%s: no callback_url", payload.TxID)
		return
	}

	body, _ := json.Marshal(payload)

	for attempt := 1; attempt <= s.maxRetry; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, agent.CallbackURL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Callback-Signature", util.Sign(payload, agentID))

		resp, err := s.httpClient.Do(req)
		httpStatus := 0
		respBody := ""
		success := false

		if err == nil {
			httpStatus = resp.StatusCode
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			respBody = buf.String()
			resp.Body.Close()
			success = httpStatus >= 200 && httpStatus < 300
		}

		// log ke DB
		_, _ = s.db.ExecContext(ctx,
			`INSERT INTO callback_logs (tx_id, agent_id, url, http_status, response_body, attempt, success)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			payload.TxID, agentID, agent.CallbackURL, httpStatus, respBody, attempt, success)

		if success {
			s.txRepo.MarkCallback(ctx, payload.TxID, "SENT", attempt)
			util.Log.Infof("[CALLBACK] OK tx=%s attempt=%d status=%d", payload.TxID, attempt, httpStatus)
			return
		}

		util.Log.Warnf("[CALLBACK] FAIL tx=%s attempt=%d err=%v", payload.TxID, attempt, err)
		time.Sleep(time.Duration(attempt*2) * time.Second) // backoff
	}

	s.txRepo.MarkCallback(ctx, payload.TxID, "FAILED", s.maxRetry)
	util.Log.Errorf("[CALLBACK] gave up tx=%s after %d attempts", payload.TxID, s.maxRetry)
}

// helper signature
var _ = fmt.Sprintf
