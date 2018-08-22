# godbc
--
    import "."

Package godbc implements deathbycaptcha's API

            func main() {
    			client := godbc.DefaultClient(`user`, `password`)

    			status, err := client.Status()
    			if err != nil {
    				panic(err)
    			}
                if status.IsServiceOverloaded {
    				fmt.Println("Service is overloaded, this may fail")
    			}

    			user, err := client.User()
    			if err != nil {
    				panic(err)
    			}
    			if user.IsBanned || !user.HasCreditLeft() {
    				panic("User is banned or no credit left")
    			}

    			res, err := client.CaptchaFromFile(`./captcha.jpg`)
    			if err != nil {
    				panic(err)
    			}
    			resolved, err := client.WaitCaptcha(res)
    			if err != nil {
    				panic(err)
    			}

    			fmt.Printf("Captcha text: %s\n", resolved.Text)
    		}

## Usage

```go
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
```
Variables Error codes returned by failures to do API calls

#### type CaptchaResponse

```go
type CaptchaResponse struct {
	ID        int64  `json:"captcha"`
	IsCorrect bool   `json:"is_correct"`
	Text      string `json:"text"`
	Status    int    `json:"status"`
	Error     string `json:"error"`
}
```

CaptchaResponse is returned as API response for all captcha related calls

#### type Client

```go
type Client struct {
	HTTPClient *http.Client
}
```

Client is the DBC client main struct

#### func  DefaultClient

```go
func DefaultClient(username, password string) *Client
```
DefaultClient returns a DBC client with default options:

    Endpoint: http://api.dbcapi.me/api/
    HttpTimeout: 30 seconds
    TLSHandshakeTimeout: 5 seconds
    CaptchaRetries: 10

#### func  NewClient

```go
func NewClient(username, password string, options *ClientOptions) *Client
```
NewClient returns a DBC client. Options not specified will take default values,
see DefaultClient

#### func (*Client) Captcha

```go
func (c *Client) Captcha(content []byte) (*CaptchaResponse, error)
```
Captcha will make a captcha call from a byte slice

#### func (*Client) CaptchaFromFile

```go
func (c *Client) CaptchaFromFile(filepath string) (*CaptchaResponse, error)
```
CaptchaFromFile will make a captcha call from a file on disk

#### func (*Client) CaptchaFromHTTPRequest

```go
func (c *Client) CaptchaFromHTTPRequest(request *http.Request) (*CaptchaResponse, error)
```
CaptchaFromHTTPRequest will make a captcha call from an http request

#### func (*Client) CaptchaFromURL

```go
func (c *Client) CaptchaFromURL(url string) (*CaptchaResponse, error)
```
CaptchaFromURL will make a captcha call from an image url

#### func (*Client) PollCaptcha

```go
func (c *Client) PollCaptcha(ressource *CaptchaResponse) (*CaptchaResponse, error)
```
PollCaptcha will make a captcha poll call

#### func (*Client) ReportCaptcha

```go
func (c *Client) ReportCaptcha(ressource *CaptchaResponse) (*CaptchaResponse, error)
```
ReportCaptcha will report a captcha as incorrectly solved

#### func (*Client) Status

```go
func (c *Client) Status() (*StatusResponse, error)
```
Status will retrieve status information

#### func (*Client) User

```go
func (c *Client) User() (*UserResponse, error)
```
User will retrieve user information

#### func (*Client) WaitCaptcha

```go
func (c *Client) WaitCaptcha(ressource *CaptchaResponse) (*CaptchaResponse, error)
```
WaitCaptcha will wait for a captcha to be solved

#### type ClientOptions

```go
type ClientOptions struct {
	Endpoint            *url.URL
	HTTPTimeout         *time.Duration
	TLSHandshakeTimeout *time.Duration
	CaptchaRetries      int
}
```

ClientOptions is the client's options struct to be sent in the constructor

#### type StatusResponse

```go
type StatusResponse struct {
	TodaysAccuracy      float64 `json:"todays_accuracy"`
	SolvedIn            float64 `json:"solved_in"`
	IsServiceOverloaded bool    `json:"is_service_overloaded"`
	Status              int     `json:"status"`
	Error               string  `json:"error"`
}
```

StatusResponse is returned as API response for the `status` call

#### type UserResponse

```go
type UserResponse struct {
	ID       int64   `json:"user"`
	Rate     float64 `json:"rate"`
	Balance  float64 `json:"balance"`
	IsBanned bool    `json:"is_banned"`
	Status   int     `json:"status"`
	Error    string  `json:"error"`
}
```

UserResponse is returned as API response for the `user` call

#### func (*UserResponse) HasCreditLeft

```go
func (u *UserResponse) HasCreditLeft() bool
```
HasCreditLeft returns true is user has enough credit to solve one captcha
