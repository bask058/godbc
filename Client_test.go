package godbc

import (
	"fmt"
)

func Example() {
	client := DefaultClient(`user`, `password`)

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

	ressource, err := client.RecaptchaWithoutProxy(`http://test.com/path_with_recaptcha`, `6Le-wvkSAAAAAPBMRTvw0Q4Muexq9bi0DJwx_mJ-`)
	resolved, err = client.WaitCaptcha(ressource)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Captcha token: %s\n", resolved.Text)
}
