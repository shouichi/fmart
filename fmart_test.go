package fmart

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIssueInvoiceParams(t *testing.T) {
	x := 5
	p := &IssueInvoiceParams{}
	errs := p.Errors()
	if y := len(errs); y != x {
		t.Errorf("expected %d errors, got: %d", x, y)
	}
}

func TestIssueInvoice(t *testing.T) {
	var res string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, res)
	}))
	defer ts.Close()

	APIEndpoint = ts.URL

	res = "not-reached"
	p := &IssueInvoiceParams{}
	id, err := IssueInvoice(p)
	if err != ErrInvalidParams {
		t.Errorf("expected ErrInvalidParams error, got: nil")
	}

	res = "invoice-1234"
	p = &IssueInvoiceParams{
		Name:         "松本行弘",
		NameKatakana: "マツモトヒロユキ",
		PhoneNumber:  "0120-444-444",
		Amount:       100,
		Expiry:       time.Now().AddDate(0, 0, 1),
	}
	id, err = IssueInvoice(p)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if id != res {
		t.Errorf("expected %s error, got: %s", res, id)
	}

	res = "-1\nerror message"
	id, err = IssueInvoice(p)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if id != "" {
		t.Errorf("expected empty id, got: %s", id)
	}
}
