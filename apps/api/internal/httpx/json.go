package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const MaxJSONBodyBytes int64 = 1 << 20

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

func DecodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, MaxJSONBodyBytes+1))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.InputOffset() > MaxJSONBodyBytes {
		return fmt.Errorf("request body too large")
	}
	var extra any
	if err := decoder.Decode(&extra); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if extra != nil {
		return fmt.Errorf("request body must contain a single JSON value")
	}
	return nil
}
