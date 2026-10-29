package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
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
	if err != nil {
		util.Log.Errorf("[CALLBACK] cannot load agent=%d for tx=%s: %v", agentID, payload.TxID, err)
		return
	}
	if agent.CallbackURL == "" {
		util.Log.Debugf("[CALLBACK] skip tx=%s: no callback_url", payload.TxID)
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		util.Log.Errorf("[CALLBACK] marshal payload error tx=%s: %v", payload.TxID, err)
		return
	}

	for attempt := 1; attempt <= s.maxRetry; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, agent.CallbackURL, bytes.NewReader(body))
		if err != nil {
			util.Log.Errorf("[CALLBACK] invalid callback request tx=%s: %v", payload.TxID, err)
			if markErr := s.txRepo.MarkCallback(ctx, payload.TxID, "FAILED", attempt); markErr != nil {
				util.Log.Errorf("[CALLBACK] failed to mark callback tx=%s: %v", payload.TxID, markErr)
			}
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		httpStatus := 0
		respBody := ""
		success := false

		if err == nil {
			httpStatus = resp.StatusCode
			responseBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
			closeErr := resp.Body.Close()
			respBody = string(responseBytes)
			if readErr != nil {
				err = readErr
			} else if closeErr != nil {
				err = closeErr
			} else {
				success = httpStatus >= 200 && httpStatus < 300
			}
		}

		// log ke DB
		if _, logErr := s.db.ExecContext(ctx,
			`INSERT INTO callback_logs (tx_id, agent_id, url, http_status, response_body, attempt, success)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			payload.TxID, agentID, agent.CallbackURL, httpStatus, respBody, attempt, success); logErr != nil {
			util.Log.Errorf("[CALLBACK] failed to write callback log tx=%s attempt=%d: %v", payload.TxID, attempt, logErr)
		}

		if success {
			if err := s.txRepo.MarkCallback(ctx, payload.TxID, "SENT", attempt); err != nil {
				util.Log.Errorf("[CALLBACK] failed to mark callback sent tx=%s: %v", payload.TxID, err)
			}
			util.Log.Infof("[CALLBACK] OK tx=%s attempt=%d status=%d", payload.TxID, attempt, httpStatus)
			return
		}

		util.Log.Warnf("[CALLBACK] FAIL tx=%s attempt=%d err=%v", payload.TxID, attempt, err)
		if attempt < s.maxRetry {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}

	if err := s.txRepo.MarkCallback(ctx, payload.TxID, "FAILED", s.maxRetry); err != nil {
		util.Log.Errorf("[CALLBACK] failed to mark callback failure tx=%s: %v", payload.TxID, err)
	}
	util.Log.Errorf("[CALLBACK] gave up tx=%s after %d attempts", payload.TxID, s.maxRetry)
}
