package fmart

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIssueInvoice(t *testing.T) {
	var res string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, res)
	}))
	defer ts.Close()

	APIEndpoint = ts.URL

	res = "invoice-1234"
	p := &IssueInvoiceParams{}
	id, err := IssueInvoice(p)
	if err != nil {
		t.Errorf("expected nii error, got: %v", err)
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
