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
)

var (
	// APIEndpoint is URL of FamilyMart Invoice API.
	APIEndpoint = "https://"
	// UserID is ID of the invoice issuer.
	UserID = ""
	// UserPassword is password of the invoice issuer.
	UserPassword = ""
)

var phoneNumberRegexp = regexp.MustCompile(`\d{2,5}-\d{2,5}-\d{3,4}`)

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

	applyValidations(errs, "name", p.Name, []validateFunc{
		validateMinLength(1),
		validateMaxLength(40),
	})

	applyValidations(errs, "name_katakana", p.NameKatakana, []validateFunc{
		validateMinLength(1),
		validateMaxLength(30),
	})

	applyValidations(errs, "phone_number", p.PhoneNumber, []validateFunc{
		validateMinLength(1),
		validateMaxLength(13),
		func(p interface{}) string {
			if !phoneNumberRegexp.MatchString(p.(string)) {
				return "invalid format"
			}
			return ""
		},
	})

	applyValidations(errs, "amount", p.Amount, []validateFunc{
		func(x interface{}) string {
			if x.(int) < 1 {
				return "must be greater than 0"
			}
			return ""
		},
		func(x interface{}) string {
			if x.(int) > 999999 {
				return "must be less than 999999"
			}
			return ""
		},
	})

	applyValidations(errs, "expiry", p.Expiry, []validateFunc{
		func(t interface{}) string {
			if t.(time.Time).Before(time.Now()) {
				return "must be future"
			}
			return ""
		},
		func(t interface{}) string {
			if t.(time.Time).After(time.Now().AddDate(0, 0, 60)) {
				return "must be within 60 days"
			}
			return ""
		},
	})

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

// ModifyInvoice takes ID of existing invoice and modifies it.
func ModifyInvoice(ID string) error {
	v := url.Values{}

	_, err := request(v)
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

// GetInvoiceStatus takes ID of existing invoice and returns its status.
func GetInvoiceStatus(ID string) error {
	v := url.Values{}

	_, err := request(v)
	return err
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
