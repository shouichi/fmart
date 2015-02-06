package fmart

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

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

func ExampleIssueInvoice() {
	p := &IssueInvoiceParams{}
	if !p.IsValid() {
	}

	id, err := IssueInvoice(p)
	if err != nil {
	}
	fmt.Println(id)
}

func TestModifyInvoice(t *testing.T) {
	var res string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, res)
	}))
	defer ts.Close()

	APIEndpoint = ts.URL

	res = "not-reached"
	p := &ModifyInvoiceParams{}
	err := ModifyInvoice(p)
	if err != ErrInvalidParams {
		t.Errorf("expected ErrInvalidParams error, got: nil")
	}

	res = "invoice-1234"
	p = &ModifyInvoiceParams{
		ID:           res,
		Name:         "松本行弘",
		NameKatakana: "マツモトヒロユキ",
		PhoneNumber:  "0120-444-444",
		Amount:       100,
		Expiry:       time.Now().AddDate(0, 0, 1),
	}
	err = ModifyInvoice(p)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	res = "-1\nerror message"
	err = ModifyInvoice(p)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func ExampleModifyInvoice() {
	p := &ModifyInvoiceParams{}
	if !p.IsValid() {
	}

	err := ModifyInvoice(p)
	if err != nil {
	}
}

func TestCancelInvoice(t *testing.T) {
	var res string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, res)
	}))
	defer ts.Close()

	APIEndpoint = ts.URL

	res = "invoice-1234"
	err := CancelInvoice(res)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	res = "-1\nerror message"
	err = CancelInvoice(res)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func ExampleCancelInvoice() {
	err := CancelInvoice("invoice-123")
	if err != nil {
	}
}

func TestAckInvoiceStatuses(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)

		e := "1\r\n2\r\n3"
		a := string(b)
		if e != a {
			t.Errorf("expected %v, got: %v", e, a)
		}
	}))
	defer ts.Close()

	APIEndpoint = ts.URL

	err := AckInvoiceStatuses([]string{"1", "2", "3"})
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

func ExampleAckInvoiceStatuses() {
	err := AckInvoiceStatuses([]string{"invoice-1", "invoice-2"})
	if err != nil {
	}
}

func TestParseInvoiceStatuses(t *testing.T) {
	var assertion func([]*InvoiceStatus, error)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertion(ParseInvoiceStatuses(r))
	}))
	defer ts.Close()

	assertion = func(statuses []*InvoiceStatus, err error) {
		if err != ErrUnauthorizedRequest {
			t.Errorf("expected ErrUnauthorizedRequest error, got nil")
		}
	}
	http.PostForm(ts.URL, url.Values{
		"login_user_id":  {"invalid_user_id"},
		"login_password": {"invalid_password"},
	})

	assertion = func(statuses []*InvoiceStatus, err error) {
		if err != ErrInvalidRequest {
			t.Errorf("expected ErrInvalidRequest error, got nil")
		}
	}
	http.PostForm(ts.URL, url.Values{
		"login_user_id":     {""},
		"login_password":    {""},
		"number_of_notify":  {"1"},
		"receipt_no_0000":   {"1"},
		"status_0000":       {"4"},
		"receipt_date_0000": {"201502082012"},
		"payment_0000":      {"100"},
	})

	assertion = func(statuses []*InvoiceStatus, err error) {
		if err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}

		if n := len(statuses); n != 3 {
			t.Errorf("expected to get 3 statuses, got: %v", n)
		}
	}
	http.PostForm(ts.URL, url.Values{
		"login_user_id":     {""},
		"login_password":    {""},
		"number_of_notify":  {"3"},
		"receipt_no_0000":   {"invlice-1"},
		"status_0000":       {"1"},
		"receipt_date_0000": {"201502082010"},
		"payment_0000":      {"101"},
		"receipt_no_0001":   {"invlice-2"},
		"status_0001":       {"2"},
		"receipt_date_0001": {"201502082010"},
		"payment_0001":      {"102"},
		"receipt_no_0002":   {"invlice-3"},
		"status_0002":       {"3"},
		"receipt_date_0002": {"201502082010"},
		"payment_0002":      {"103"},
	})
}

func ExampleParseInvoiceStatuses() {
	statuses, err := ParseInvoiceStatuses(&http.Request{})
	if err != nil {
	}

	for _, s := range statuses {
		switch s.Status {
		case StatusDepositMade:
			// do something
		case StatusDepositCanceled:
			// do another thing
		case StatusDepositFinalized:
			// do yet another thing
		default:
			// not reached
		}
	}
}
