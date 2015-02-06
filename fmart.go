// Package fmart provides FamilyMart invoice API client. It converts character
// encodings, validates parameters and handles errors.
package fmart

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var (
	// ErrInvalidParams is returned when params are invalid.
	ErrInvalidParams = errors.New("fmart: invalid params")
)

var (
	apiEndpoint  = "https://"
	userID       = ""
	userPassword = ""
)

// IssueInvoiceParams represents params for IssueInvoice and provides validations.
type IssueInvoiceParams struct {
	Name         string
	NameKatakana string
	PhoneNumber  string
	Amount       int
	Expiry       time.Time
}

// IsValid returns true iff all values are valid.
func (p *IssueInvoiceParams) IsValid() bool {
	return true
}

// IssueInvoice issues a new invoice. Returns invoice identifier when success.
func IssueInvoice(p *IssueInvoiceParams) (string, error) {
	if !p.IsValid() {
		return "", ErrInvalidParams
	}

	t := p.Expiry
	v := url.Values{
		"login_user_id":  {userID},
		"login_password": {userPassword},
		"regist_type":    {"1"},
		"name":           {p.Name},
		"kana":           {p.NameKatakana},
		"phone_no":       {p.PhoneNumber},
		"payment":        {strconv.Itoa(p.Amount)},
		"date_of_expiry": {fmt.Sprintf("%04d%02d%02d", t.Year(), t.Month(), t.Day())},
	}

	return request(v)
}

// ModifyInvoice takes ID of existing invoice and modifies it.
func ModifyInvoice(ID string) error {
	v := url.Values{}

	_, err := request(v)
	return err
}

// CancelInvoice takes ID of existing invoice and cancels it.
func CancelInvoice(ID string) error {
	v := url.Values{}

	_, err := request(v)
	return err
}

// GetInvoiceStatus takes ID of existing invoice and returns its status.
func GetInvoiceStatus(ID string) error {
	v := url.Values{}

	_, err := request(v)
	return err
}

func request(p url.Values) (string, error) {
	e := encodeShiftJIS(strings.NewReader(p.Encode()))
	res, err := http.Post(apiEndpoint, "application/x-www-form-urlencoded", e)
	if err != nil {
		return "", fmt.Errorf("fmart: %v", err)
	}

	if res.StatusCode != 200 {
		return "", errors.New("fmart: server returned non 200")
	}

	body, err := ioutil.ReadAll(decodeShiftJIS(res.Body))
	if err != nil {
		return "", errors.New("fmart: could not read response body")
	}

	lines := strings.Split(string(body), "\n")
	if len(lines) == 1 {
		return lines[0], nil
	}
	return "", fmt.Errorf("fmart: %s", string(body))
}

func encodeShiftJIS(r io.Reader) io.Reader {
	return transform.NewReader(r, japanese.ShiftJIS.NewEncoder())
}

func decodeShiftJIS(r io.Reader) io.Reader {
	return transform.NewReader(r, japanese.ShiftJIS.NewDecoder())
}
