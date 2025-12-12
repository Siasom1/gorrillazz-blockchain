package rpc

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleGetMerchantPayments(w http.ResponseWriter, r *http.Request) {
	merchant := r.URL.Query().Get("merchant")

	if merchant == "" {
		http.Error(w, "missing merchant", http.StatusBadRequest)
		return
	}

	// TODO: replace this with real DB/chain lookup
	payments := s.PaymentsDB.GetPaymentsByMerchant(merchant)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"payments": payments,
	})
}
