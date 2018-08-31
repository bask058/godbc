/*
Package godbc implements deathbycaptcha's API
*/
package godbc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

//Error codes returned by failures to do API calls
var (
	//ErrCredentialsRejected - Credentials were rejected by service
	ErrCredentialsRejected = errors.New("Credentials were rejected")
	//ErrInvalidFormat - Only image formats accepted are JPG, PNG, GIF, BMP
	ErrInvalidFormat = errors.New("Content is not in an accepted format (JPG, PNG, GIF, BMP)")
	//ErrContentTooBig - As we use raw data, the size limit for an image is 180KB. With b64 encoded data, the limit woulf be 120KB
	ErrContentTooBig = errors.New("Content is too big (180KB max)")
	//ErrCaptchaTimeout - The captcha was not resolved under a good amount of time - give up
	ErrCaptchaTimeout = errors.New("Captcha query has timed out")
	//ErrCaptchaRejected - The server did reject the image data, as not being a valid image
	ErrCaptchaRejected = errors.New("Captcha was rejected - not a valid image")
	//ErrCaptchaInvalid - The captcha was not correctly solved
	ErrCaptchaInvalid = errors.New("Captcha is invalid")
	//ErrUnexpectedServerError - An unexpected error occured on the server side
	ErrUnexpectedServerError = errors.New("Unexpected error on server side")
	//ErrUnexpectedServerResponse - We got an unexpected response from the server
	ErrUnexpectedServerResponse = errors.New("Unexpected output from server")
	//ErrOverloadedServer - Server is overloaded and cannot take our query
	ErrOverloadedServer = errors.New("Server is overloaded - try again later")
	//ErrReportRejected - Our captcha reporting was rejected, either the captcha id is incorrect, our user is banned or we are reporting it too late (1 hour max)
	ErrReportRejected = errors.New("Report was rejected - Bad captcha id, user banned or captcha too old (1h max)")
	//ErrCaptchaDoesNotExist - The captcha id provided is non-existent
	ErrCaptchaDoesNotExist = errors.New("Captcha does not exist")
)

//Recaptcha by token proxy types
const (
	//RecaptchaProxyTypeHTTP - HTTP proxy type
	RecaptchaProxyTypeHTTP = "HTTP"
)

//Client is the DBC client main struct
type Client struct {
	HTTPClient *http.Client
	username   string
	password   string
	options    *ClientOptions
}

//ClientOptions is the client's options struct to be sent in the constructor
type ClientOptions struct {
	Endpoint            *url.URL
	HTTPTimeout         *time.Duration
	TLSHandshakeTimeout *time.Duration
	CaptchaRetries      int
}

//CaptchaResponse is returned as API response for all captcha related calls
type CaptchaResponse struct {
	ID        int64  `json:"captcha"`
	IsCorrect bool   `json:"is_correct"`
	Text      string `json:"text"`
	Status    int    `json:"status"`
	Error     string `json:"error"`
}

//RecaptchaRequestPayload is a payload that goes in a request for recaptcha by token api
type RecaptchaRequestPayload struct {
	PageURL   string `json:"pageurl"`
	GoogleKey string `json:"googlekey"`
	Proxy     string `json:"proxy,omitempty"`
	ProxyType string `json:"proxytype,omitempty"`
}

//StatusResponse  is returned as API response for the `status` call
type StatusResponse struct {
	TodaysAccuracy      float64 `json:"todays_accuracy"`
	SolvedIn            float64 `json:"solved_in"`
	IsServiceOverloaded bool    `json:"is_service_overloaded"`
	Status              int     `json:"status"`
	Error               string  `json:"error"`
}

//UserResponse  is returned as API response for the `user` call
type UserResponse struct {
	ID       int64   `json:"user"`
	Rate     float64 `json:"rate"`
	Balance  float64 `json:"balance"`
	IsBanned bool    `json:"is_banned"`
	Status   int     `json:"status"`
	Error    string  `json:"error"`
}

//HasCreditLeft returns true is user has enough credit to solve one captcha
func (u *UserResponse) HasCreditLeft() bool {
	if u.Rate == 0 {
		return true
	}

	return u.Balance/u.Rate >= 1
}

/*DefaultClient returns a DBC client with default options:

  Endpoint: http://api.dbcapi.me/api/
  HttpTimeout: 30 seconds
  TLSHandshakeTimeout: 5 seconds
  CaptchaRetries: 10
*/
func DefaultClient(username, password string) *Client {
	return NewClient(username, password, setDefaultOptions(nil))
}

//NewClient returns a DBC client. Options not specified will take default values, see DefaultClient
func NewClient(username, password string, options *ClientOptions) *Client {
	options = setDefaultOptions(options)
	return &Client{
		HTTPClient: &http.Client{
			Timeout: *options.HTTPTimeout,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: *options.HTTPTimeout,
				}).Dial,
				TLSHandshakeTimeout: *options.TLSHandshakeTimeout,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		},
		username: username,
		password: password,
		options:  options,
	}
}

func setDefaultOptions(options *ClientOptions) *ClientOptions {
	newOptions := &ClientOptions{}

	if options == nil {
		options = &ClientOptions{}
	}

	if options.Endpoint == nil {
		endpoint, _ := url.Parse(`http://api.dbcapi.me/api/`)
		newOptions.Endpoint = endpoint
	} else {
		newOptions.Endpoint = options.Endpoint
	}

	if options.HTTPTimeout == nil {
		d := time.Second * 30
		newOptions.HTTPTimeout = &d
	} else {
		newOptions.HTTPTimeout = options.HTTPTimeout
	}

	if options.TLSHandshakeTimeout == nil {
		d := time.Second * 5
		newOptions.TLSHandshakeTimeout = &d
	} else {
		newOptions.TLSHandshakeTimeout = options.TLSHandshakeTimeout
	}

	if options.CaptchaRetries < 1 {
		newOptions.CaptchaRetries = 30
	} else {
		newOptions.CaptchaRetries = options.CaptchaRetries
	}

	return newOptions
}

//CaptchaFromURL will make a captcha call from an image url
func (c *Client) CaptchaFromURL(url string) (*CaptchaResponse, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.CaptchaFromHTTPRequest(request)
}

//CaptchaFromHTTPRequest will make a captcha call from an http request
func (c *Client) CaptchaFromHTTPRequest(request *http.Request) (*CaptchaResponse, error) {
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return c.Captcha(body)
}

//CaptchaFromFile will make a captcha call from a file on disk
func (c *Client) CaptchaFromFile(filepath string) (*CaptchaResponse, error) {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	return c.Captcha(content)
}

//Captcha will make a captcha call from a byte slice
func (c *Client) Captcha(content []byte) (*CaptchaResponse, error) {
	if !c.isValidFormat(content) {
		return nil, ErrInvalidFormat
	}

	urlReq, err := c.options.Endpoint.Parse(`captcha`)
	if err != nil {
		return nil, err
	}

	postBody := &bytes.Buffer{}
	writer := multipart.NewWriter(postBody)
	err = writer.WriteField("username", c.username)
	if err != nil {
		return nil, err
	}
	err = writer.WriteField("password", c.password)
	if err != nil {
		return nil, err
	}
	w, err := writer.CreateFormFile("captchafile", "captcha")
	if err != nil {
		return nil, err
	}
	_, err = w.Write(content)
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(`POST`, urlReq.String(), postBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	body, err := c.makeRequest(req)
	response := &CaptchaResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, ErrUnexpectedServerResponse
	}
	if response.Status == 255 {
		return nil, fmt.Errorf("Generic error from service: %s", response.Error)
	}

	return response, nil
}

/*RecaptchaWithoutProxy will make a recaptcha by token call, without providing a proxy
  pageurl: the url of the webpage with the challenge
  googlekey: the google data-sitekey token
*/
func (c *Client) RecaptchaWithoutProxy(pageurl, googlekey string) (*CaptchaResponse, error) {
	return c.Recaptcha(pageurl, googlekey, "", "")
}

/*Recaptcha will make a recaptcha by token call
  pageurl: the url of the webpage with the challenge
  googlekey: the google data-sitekey token
  proxy: address of the proxy
  proxyType: type of the proxy
*/
func (c *Client) Recaptcha(pageurl, googlekey, proxy, proxyType string) (*CaptchaResponse, error) {
	urlReq, err := c.options.Endpoint.Parse(`captcha`)
	if err != nil {
		return nil, err
	}

	v := url.Values{}
	v.Set("username", c.username)
	v.Set("password", c.password)
	v.Set("type", "4")

	payload := RecaptchaRequestPayload{
		PageURL:   pageurl,
		GoogleKey: googlekey,
	}

	if proxy != "" {
		payload.Proxy = proxy
		if proxyType == "" {
			payload.ProxyType = RecaptchaProxyTypeHTTP
		} else {
			payload.ProxyType = proxyType
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	v.Set("token_params", string(payloadBytes))

	req, err := http.NewRequest(`POST`, urlReq.String(), strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	body, err := c.makeRequest(req)
	response := &CaptchaResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, ErrUnexpectedServerResponse
	}
	if response.Status == 255 {
		return nil, fmt.Errorf("Generic error from service: %s", response.Error)
	}

	return response, nil
}

//PollCaptcha will make a captcha poll call
func (c *Client) PollCaptcha(ressource *CaptchaResponse) (*CaptchaResponse, error) {
	urlReq, err := c.options.Endpoint.Parse(fmt.Sprintf(`captcha/%d`, ressource.ID))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(`GET`, urlReq.String(), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.makeRequest(req)
	response := &CaptchaResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, ErrUnexpectedServerResponse
	}
	if response.Status == 255 {
		return nil, fmt.Errorf("Generic error from service: %s", response.Error)
	}

	if response.Text == "?" {
		return nil, ErrCaptchaInvalid
	}

	return response, nil
}

//WaitCaptcha will wait for a captcha to be solved
func (c *Client) WaitCaptcha(ressource *CaptchaResponse) (*CaptchaResponse, error) {
	for i := 1; i <= c.options.CaptchaRetries; i++ {
		time.Sleep(time.Duration(i) * time.Second)
		response, err := c.PollCaptcha(ressource)
		if err != nil {
			if err == ErrCaptchaInvalid {
				return nil, err
			}
			continue
		}
		if response.IsCorrect && response.Text != "" {
			return response, nil
		}
	}
	return nil, ErrCaptchaTimeout
}

//ReportCaptcha will report a captcha as incorrectly solved
func (c *Client) ReportCaptcha(ressource *CaptchaResponse) (*CaptchaResponse, error) {
	urlReq, err := c.options.Endpoint.Parse(fmt.Sprintf(`captcha/%d/report`, ressource.ID))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(`GET`, urlReq.String(), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.makeRequest(req)
	response := &CaptchaResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, ErrUnexpectedServerResponse
	}
	if response.Status == 255 {
		return nil, fmt.Errorf("Generic error from service: %s", response.Error)
	}

	return response, nil
}

//User will retrieve user information
func (c *Client) User() (*UserResponse, error) {
	urlReq, err := c.options.Endpoint.Parse(`user`)
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	v.Set("username", c.username)
	v.Set("password", c.password)
	urlReq.RawQuery = v.Encode()
	req, err := http.NewRequest(`GET`, urlReq.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	body, err := c.makeRequest(req)
	response := &UserResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, ErrUnexpectedServerResponse
	}
	if response.Status == 255 {
		return nil, fmt.Errorf("Generic error from service: %s", response.Error)
	}

	return response, nil
}

//Status will retrieve status information
func (c *Client) Status() (*StatusResponse, error) {
	urlReq, err := c.options.Endpoint.Parse(`status`)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(`GET`, urlReq.String(), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.makeRequest(req)
	response := &StatusResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, ErrUnexpectedServerResponse
	}
	if response.Status == 255 {
		return nil, fmt.Errorf("Generic error from service: %s", response.Error)
	}

	return response, nil
}

func (c *Client) makeRequest(request *http.Request) ([]byte, error) {
	request.Header.Add(`Accept`, `application/json`)
	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, ErrCredentialsRejected
	}
	if resp.StatusCode == 400 {
		return nil, ErrCaptchaRejected
	}
	if resp.StatusCode == 500 {
		return nil, ErrUnexpectedServerError
	}
	if resp.StatusCode == 503 {
		if regexp.MustCompile(`captcha$`).MatchString(request.URL.Path) {
			return nil, ErrOverloadedServer
		}
		if regexp.MustCompile(`report$`).MatchString(request.URL.Path) {
			return nil, ErrReportRejected
		}
		return nil, ErrUnexpectedServerResponse
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *Client) isValidFormat(content []byte) bool {
	if bytes.Compare(content[0:3], []byte{255, 216, 255}) == 0 /*jpg*/ || bytes.Compare(content[0:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) == 0 /*png*/ || bytes.Compare(content[0:3], []byte{71, 73, 70}) == 0 /*gif*/ || bytes.Compare(content[0:2], []byte{66, 77}) == 0 /*bmp*/ {
		return true
	}
	return false
}
