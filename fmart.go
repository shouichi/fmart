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
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var (
	// ErrInvalidParams is returned when params are invalid.
	ErrInvalidParams = errors.New("fmart: invalid params")
	// ErrUnauthorizedRequest is returned when request contains invalid id or password.
	ErrUnauthorizedRequest = errors.New("fmart: invalid params")
	// ErrInvalidRequest is returned when request contains invalid data.
	ErrInvalidRequest = errors.New("fmart: invalid params")
)

var (
	// APIEndpoint is URL of FamilyMart Invoice API.
	APIEndpoint = "https://"
	// UserID is ID of the invoice issuer.
	UserID = ""
	// UserPassword is password of the invoice issuer.
	UserPassword = ""
)

var (
	idValidations = []validateFunc{
		validateMinLength(1),
		validateMaxLength(18),
	}

	nameValidations = []validateFunc{
		validateMinLength(1),
		validateMaxLength(40),
	}

	nameKatakanaValidations = []validateFunc{
		validateMinLength(1),
		validateMaxLength(30),
	}

	phoneNumberValidations = []validateFunc{
		validateMinLength(1),
		validateMaxLength(13),
		validateFormat(regexp.MustCompile(`\d{2,5}-\d{2,5}-\d{3,4}`)),
	}

	amountValidations = []validateFunc{
		validateMin(1),
		validateMax(999999),
	}

	expiryValidations = []validateFunc{
		validateMinTime(time.Now()),
		validateMaxTime(time.Now().AddDate(0, 0, 60)),
	}
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
	return len(p.Errors()) == 0
}

// Errors returns map of errors where key is the invalid field and value is
// array of error messages.
func (p *IssueInvoiceParams) Errors() map[string][]string {
	errs := make(map[string][]string)

	applyValidations(errs, "name", p.Name, nameValidations)
	applyValidations(errs, "name_katakana", p.NameKatakana, nameKatakanaValidations)
	applyValidations(errs, "phone_number", p.PhoneNumber, phoneNumberValidations)
	applyValidations(errs, "amount", p.Amount, amountValidations)
	applyValidations(errs, "expiry", p.Expiry, expiryValidations)

	return errs
}

// Params returns url.Values representation of IssueInvoiceParams.
func (p *IssueInvoiceParams) Params() url.Values {
	return url.Values{
		"login_user_id":  {UserID},
		"login_password": {UserPassword},
		"regist_type":    {"1"},
		"name":           {p.Name},
		"kana":           {p.NameKatakana},
		"phone_no":       {p.PhoneNumber},
		"payment":        {strconv.Itoa(p.Amount)},
		"date_of_expiry": {formatTime(p.Expiry)},
	}
}

// IssueInvoice issues a new invoice. Returns invoice identifier when success.
func IssueInvoice(p *IssueInvoiceParams) (string, error) {
	if !p.IsValid() {
		return "", ErrInvalidParams
	}

	return request(p.Params())
}

// ModifyInvoiceParams represents params for ModifyInvoice and provides validations.
type ModifyInvoiceParams struct {
	ID           string
	Name         string
	NameKatakana string
	PhoneNumber  string
	Amount       int
	Expiry       time.Time
}

// IsValid returns true iff all values are valid.
func (p *ModifyInvoiceParams) IsValid() bool {
	return len(p.Errors()) == 0
}

// Errors returns map of errors where key is the invalid field and value is
// array of error messages.
func (p *ModifyInvoiceParams) Errors() map[string][]string {
	errs := make(map[string][]string)

	applyValidations(errs, "id", p.ID, idValidations)
	applyValidations(errs, "name", p.Name, nameValidations)
	applyValidations(errs, "name_katakana", p.NameKatakana, nameKatakanaValidations)
	applyValidations(errs, "phone_number", p.PhoneNumber, phoneNumberValidations)
	applyValidations(errs, "amount", p.Amount, amountValidations)
	applyValidations(errs, "expiry", p.Expiry, expiryValidations)

	return errs
}

// Params returns url.Values representation of ModifyInvoiceParams.
func (p *ModifyInvoiceParams) Params() url.Values {
	return url.Values{
		"login_user_id":  {UserID},
		"login_password": {UserPassword},
		"regist_type":    {"2"},
		"receipt_no":     {p.ID},
		"name":           {p.Name},
		"kana":           {p.NameKatakana},
		"phone_no":       {p.PhoneNumber},
		"payment":        {strconv.Itoa(p.Amount)},
		"date_of_expiry": {formatTime(p.Expiry)},
	}
}

// ModifyInvoice takes ID of existing invoice and modifies it.
func ModifyInvoice(p *ModifyInvoiceParams) error {
	if !p.IsValid() {
		return ErrInvalidParams
	}

	_, err := request(p.Params())
	return err
}

// CancelInvoice takes ID of existing invoice and cancels it.
func CancelInvoice(ID string) error {
	v := url.Values{
		"login_user_id":  {UserID},
		"login_password": {UserPassword},
		"regist_type":    {"9"},
		"receipt_no":     {ID},
	}

	_, err := request(v)
	return err
}

// AckInvoiceStatuses takes array of invoice IDs and sends acknowledgement request.
func AckInvoiceStatuses(IDs []string) error {
	r := strings.NewReader(strings.Join(IDs, "\r\n"))
	res, err := http.Post(APIEndpoint, "text/plain", r)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return errors.New("fmart: server returned non 200")
	}

	return nil
}

const (
	// StatusDepositMade represents the situation where customer deposited but still be able to cancel.
	StatusDepositMade = 1
	// StatusDepositCanceled represents the situation where customer deposited and canceled.
	StatusDepositCanceled = 2
	// StatusDepositFinalized represents the situation where customer deposited and can't cancel.
	StatusDepositFinalized = 3
)

// InvoiceStatus represents invoice status.
type InvoiceStatus struct {
	ID        string
	Amount    int
	Status    int
	UpdatedAt time.Time
}

// ParseInvoiceStatuses takes *http.Request, parses it and returns statuses of
// existing invlices. It returns an error when one or more statuses contains
// invalid data.
func ParseInvoiceStatuses(r *http.Request) ([]*InvoiceStatus, error) {
	if r.FormValue("login_user_id") != UserID ||
		r.FormValue("login_password") != UserPassword {
		return nil, ErrUnauthorizedRequest
	}

	n, err := strconv.Atoi(r.FormValue("number_of_notify"))
	if err != nil {
		return nil, ErrInvalidRequest
	}

	statuses := make([]*InvoiceStatus, n)

	for i := 0; i < n; i++ {
		s, err := parseInvoiceStatusAt(r, i)
		if err != nil {
			return nil, ErrInvalidRequest
		}

		statuses[i] = s
	}

	return statuses, nil
}

func parseInvoiceStatusAt(r *http.Request, i int) (*InvoiceStatus, error) {
	id := r.FormValue(fmt.Sprintf("receipt_no_%04d", i))
	if id == "" {
		return nil, ErrInvalidRequest
	}

	amount, err := strconv.Atoi(r.FormValue(fmt.Sprintf("payment_%04d", i)))
	if err != nil {
		return nil, ErrInvalidRequest
	}

	var status int
	switch r.FormValue(fmt.Sprintf("status_%04d", i)) {
	case "1":
		status = StatusDepositMade
	case "2":
		status = StatusDepositCanceled
	case "3":
		status = StatusDepositFinalized
	default:
		return nil, ErrInvalidRequest
	}

	updatedAt, err := time.Parse("200601020304", r.FormValue(fmt.Sprintf("receipt_date_%04d", i)))
	if err != nil {
		return nil, ErrInvalidRequest
	}

	return &InvoiceStatus{
		ID:        id,
		Amount:    amount,
		Status:    status,
		UpdatedAt: updatedAt,
	}, nil
}

func request(p url.Values) (string, error) {
	e := encodeShiftJIS(strings.NewReader(p.Encode()))
	res, err := http.Post(APIEndpoint, "application/x-www-form-urlencoded", e)
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

func formatTime(t time.Time) string {
	return fmt.Sprintf("%04d%02d%02d", t.Year(), t.Month(), t.Day())
}

// validateFunc takes a value and returns error message if any.
type validateFunc func(v interface{}) string

func validateMin(n int) validateFunc {
	return func(x interface{}) string {
		if x.(int) < n {
			return fmt.Sprintf("must be greater than %d", n)
		}
		return ""
	}
}

func validateMax(n int) validateFunc {
	return func(x interface{}) string {
		if x.(int) > n {
			return fmt.Sprintf("must be less than %d", n)
		}
		return ""
	}
}

func validateMinLength(n int) validateFunc {
	return func(v interface{}) string {
		if len(v.(string)) < n {
			return fmt.Sprintf("must be longer than %d", n)
		}
		return ""
	}
}

func validateMaxLength(n int) validateFunc {
	return func(v interface{}) string {
		if len(v.(string)) > n {
			return fmt.Sprintf("must be less than %d", n)
		}
		return ""
	}
}

func validateFormat(r *regexp.Regexp) validateFunc {
	return func(p interface{}) string {
		if !r.MatchString(p.(string)) {
			return "invalid format"
		}
		return ""
	}
}

func validateMinTime(t time.Time) validateFunc {
	return func(v interface{}) string {
		if v.(time.Time).Before(t) {
			return fmt.Sprintf("must be after %v", t)
		}
		return ""
	}
}

func validateMaxTime(t time.Time) validateFunc {
	return func(v interface{}) string {
		if v.(time.Time).After(t) {
			return fmt.Sprintf("must be before %v", t)
		}
		return ""
	}
}

func applyValidations(m map[string][]string, k string, v interface{}, fns []validateFunc) {
	for _, fn := range fns {
		if msg := fn(v); msg != "" {
			if _, ok := m[k]; ok {
				m[k] = append(m[k], msg)
			} else {
				m[k] = []string{msg}
			}
		}
	}
}
