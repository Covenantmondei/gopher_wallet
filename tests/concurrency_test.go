package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentTransfers(t *testing.T) {
	rig := setupTestRig(t)

	senderEmail := fmt.Sprintf("sender-%d@example.com", time.Now().UnixNano())
	receiverEmail := fmt.Sprintf("receiver-%d@example.com", time.Now().UnixNano())
	pass := "password12345"

	senderToken := registerAndLogin(t, rig, senderEmail, pass)
	receiverToken := registerAndLogin(t, rig, receiverEmail, pass)

	// Fund sender with ₦10,000 (1,000,000 kobo)
	creditPayload, _ := json.Marshal(map[string]any{
		"amount":      1000000,
		"description": "Funding for concurrency test",
	})
	creditReq := httptest.NewRequest(http.MethodPost, "/api/transactions/credit", bytes.NewReader(creditPayload))
	creditReq.Header.Set("Authorization", "Bearer "+senderToken)
	creditRR := httptest.NewRecorder()
	rig.mux.ServeHTTP(creditRR, creditReq)

	if creditRR.Code != http.StatusOK {
		t.Fatalf("failed to fund sender: %s", creditRR.Body.String())
	}

	// Launch 10 concurrent transfer attempts of ₦2,000 (200,000 kobo) each.
	// Initial balance: ₦10,000.
	// Exactly 5 transfers should succeed (5 * ₦2,000 = ₦10,000) and 5 must fail with insufficient funds.
	const totalRoutines = 10
	var successCount atomic.Int32
	var failedCount atomic.Int32
	var wg sync.WaitGroup

	transferAmount := 200000 // 2000 NGN

	for i := 0; i < totalRoutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			payload, _ := json.Marshal(map[string]any{
				"recipient_email": receiverEmail,
				"amount":          transferAmount,
				"description":     fmt.Sprintf("Concurrent transfer #%d", routineID),
			})

			req := httptest.NewRequest(http.MethodPost, "/api/transfers", bytes.NewReader(payload))
			req.Header.Set("Authorization", "Bearer "+senderToken)
			rr := httptest.NewRecorder()

			rig.mux.ServeHTTP(rr, req)

			if rr.Code == http.StatusOK {
				successCount.Add(1)
			} else if rr.Code == http.StatusBadRequest {
				failedCount.Add(1)
			} else {
				t.Errorf("routine #%d returned unexpected status: %d (%s)", routineID, rr.Code, rr.Body.String())
			}
		}(i)
	}

	wg.Wait()

	if successCount.Load() != 5 {
		t.Errorf("expected exactly 5 transfers to succeed, got %d", successCount.Load())
	}
	if failedCount.Load() != 5 {
		t.Errorf("expected exactly 5 transfers to fail with insufficient funds, got %d", failedCount.Load())
	}

	// Verify final balances
	// Sender balance should be exactly 0
	balReqSender := httptest.NewRequest(http.MethodGet, "/api/wallet/balance", nil)
	balReqSender.Header.Set("Authorization", "Bearer "+senderToken)
	balRRSender := httptest.NewRecorder()
	rig.mux.ServeHTTP(balRRSender, balReqSender)

	var balSender struct {
		Balance int64 `json:"balance"`
	}
	_ = json.Unmarshal(balRRSender.Body.Bytes(), &balSender)
	if balSender.Balance != 0 {
		t.Errorf("expected sender balance to be exactly 0, got %d", balSender.Balance)
	}

	// Receiver balance should be exactly 1,000,000 kobo (₦10,000)
	balReqReceiver := httptest.NewRequest(http.MethodGet, "/api/wallet/balance", nil)
	balReqReceiver.Header.Set("Authorization", "Bearer "+receiverToken)
	balRRReceiver := httptest.NewRecorder()
	rig.mux.ServeHTTP(balRRReceiver, balReqReceiver)

	var balReceiver struct {
		Balance int64 `json:"balance"`
	}
	_ = json.Unmarshal(balRRReceiver.Body.Bytes(), &balReceiver)
	if balReceiver.Balance != 1000000 {
		t.Errorf("expected receiver balance to be exactly 1000000, got %d", balReceiver.Balance)
	}
}
