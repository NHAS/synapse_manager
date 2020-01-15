// synapse_manage project main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

type Identifier struct {
	IdentifierType string `json:"type"` //"type": "m.id.user"
	User           string `json:"user"` //  "user": "<user_id or user localpart>"

}

type Login struct {
	LoginType string     `json:"type"` //"type": "m.login.password",
	Iden      Identifier `json:"identifier"`
	Password  string     `json:"password"`
}

type AuthorisationResponse struct {
	UserID      string `json:"user_id"`
	AccessToken string `json:"access_token"`
	HomeServer  string `json:"home_server"`
	DeviceID    string `json:"device_id"`
}

type authorisationHeaderTransport struct {
	underlyingTransport http.RoundTripper
	authToken           string
}

func (t *authorisationHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.authToken)
	return t.underlyingTransport.RoundTrip(req)
}

func login(baseURL, user, pass string) (string, error) {
	loginStruct := Login{
		LoginType: "m.login.password",
		Password:  pass,
		Iden: Identifier{
			IdentifierType: "m.id.user",
			User:           user,
		},
	}

	login_info, err := json.Marshal(loginStruct)
	if err != nil {
		return "", err
	}

	auth, err := http.Post(baseURL+"/_matrix/client/r0/login", "application/json", bytes.NewBuffer(login_info))
	if err != nil {
		return "", err
	}
	defer auth.Body.Close()

	body, err := ioutil.ReadAll(auth.Body)
	if err != nil {
		return "", err
	}

	var authResp AuthorisationResponse
	err = json.Unmarshal(body, &authResp)
	if err != nil {
		return "", err
	}

	return authResp.AccessToken, nil

}

func logout(baseURL string, client *http.Client) error {
	req, err := http.NewRequest("POST", baseURL+"/_matrix/client/r0/logout", nil)
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	return err
}

func ls(baseURL string, client *http.Client) error {
	req, err := http.NewRequest("GET", baseURL+"/_synapse/admin/v2/users", nil)
	if err != nil {
		return err
	}

	usersList, err := client.Do(req)
	if err != nil {
		return err
	}
	defer usersList.Body.Close()

	body, err := ioutil.ReadAll(usersList.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))

	return err
}

func query(baseURL, who string, client *http.Client) error {
	req, err := http.NewRequest("GET", baseURL+"/_synapse/admin/v1/whois/"+who, nil)
	if err != nil {
		return err
	}

	userActivity, err := client.Do(req)
	if err != nil {
		return err
	}
	defer userActivity.Body.Close()

	body, err := ioutil.ReadAll(userActivity.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))

	return err
}

func deactivate(baseURL, who string, client *http.Client) error {
	req, err := http.NewRequest("POST", baseURL+"/_synapse/admin/v1/deactivate/"+who, bytes.NewBuffer([]byte(`{"erase":true}`)))
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return err
	}

	deactivateResponse, err := client.Do(req)
	if err != nil {
		return err
	}
	defer deactivateResponse.Body.Close()

	body, err := ioutil.ReadAll(deactivateResponse.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))

	return err
}

func reset(baseURL, who, pass string, client *http.Client) error {

	req, err := http.NewRequest("POST", baseURL+"/_synapse/admin/v1/reset_password/"+who, bytes.NewBuffer([]byte("{\"new_password\":\""+pass+"\"}")))
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return err
	}

	resetResponse, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resetResponse.Body.Close()

	body, err := ioutil.ReadAll(resetResponse.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))

	return err
}

func getSensitive() string {
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	password := string(bytePassword)

	return strings.TrimSpace(password)
}

func main() {
	//
	serverUrl := flag.String("url", "http://localhost:8008", "The URL that points towards the matrix server")

	isDeactivate := flag.Bool("deactivate", false, "Deactivate an account, requires --target")
	isList := flag.Bool("list", false, "List all users, requires no arguments")
	isReset := flag.Bool("reset", false, "Reset users account with new password, needs --target and --pass")
	isQuery := flag.Bool("query", false, "Queries a user and gets its current information, needs --target")
	targetUser := flag.String("target", "", "The user account to be acted upon (if required)")

	flag.Parse()

	u, err := url.Parse(*serverUrl)
	if err != nil {
		log.Fatal("Please enter valid URL")
	}

	if !*isDeactivate && !*isList && !*isReset && !*isQuery {
		flag.PrintDefaults()
		log.Fatal("Please specify an option")

	}

	if len(*targetUser) == 0 && (*isDeactivate || *isReset || *isQuery) {
		flag.PrintDefaults()
		log.Fatal("You need --target to use the option")

	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Admin username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Admin password: ")
	password := getSensitive()
	fmt.Print("\n")

	serverString := u.Scheme + "://" + u.Host
	userString := "@" + *targetUser + ":" + u.Host

	token, err := login(serverString, username, password)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{Transport: &authorisationHeaderTransport{underlyingTransport: http.DefaultTransport, authToken: token}}
	defer func() {
		if err := logout(serverString, client); err != nil {
			log.Fatal(err)
		}
	}()

	if *isDeactivate {
		err = deactivate(serverString, userString, client)
	} else if *isList {
		err = ls(serverString, client)
	} else if *isQuery {
		err = query(serverString, userString, client)
	} else if *isReset {
		fmt.Print("Enter new user password for ", *targetUser, ": ")
		err = reset(serverString, userString, getSensitive(), client)
	}

	if err != nil {
		log.Fatal(err)
	}

}
