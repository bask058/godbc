package godbc

import (
	"fmt"
	"os"
	"testing"
)

var user = os.Getenv("DBCUSERNAME")
var pass = os.Getenv("DBCPASSWORD")
var client = DefaultClient(user, pass)

func TestClientStatus(t *testing.T) {
	response, err := client.Status()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(response)
}

func TestClientUser(t *testing.T) {
	user, err := client.User()
	if err != nil {
		t.Fatal(err)
	}
	if user.IsBanned || !user.HasCreditLeft() {
		fmt.Println("User is banned or no credit left")
	}
	fmt.Println(user)
}

func TestClientCaptcha(t *testing.T) {
	response, err := client.CaptchaFromURL("https://image.ibb.co/hOUgse/af8d6acd142150c2f897c19d65e186b2c40592f9.png")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(response)

	response, err = client.WaitCaptcha(response)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(response)
}
