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

	var out bytes.Buffer
	json.Indent(&out, body, "", "\t")
	fmt.Println(out.String())

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

	var out bytes.Buffer
	json.Indent(&out, body, "", "\t")
	fmt.Println(out.String())

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

	_, err = ioutil.ReadAll(resetResponse.Body)
	if err != nil {
		return err
	}

	fmt.Println("Success!")
	return nil
}

func purge(baseURL, room string, client *http.Client) error {

	req, err := http.NewRequest("POST", baseURL+"/_synapse/admin/v1/purge_room", bytes.NewBuffer([]byte("{\"room_id\":\""+room+"\"}")))
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
	return nil
}

type Room struct {
	Room_id         string
	Name            string
	Canonical_alias string
	Joined_members  int
}

type RoomsList struct {
	Offset      int
	Total_rooms int
	Rooms       []Room
}

func autopurge(baseURL string, client *http.Client) error {

	list, err := ls_room(baseURL, client)

	var roomList RoomsList
	err = json.Unmarshal([]byte(list), &roomList)
	if err != nil {
		return err
	}

	i := 0
	for _, room := range roomList.Rooms {
		if room.Joined_members == 0 || (room.Canonical_alias == "" && room.Joined_members > 2) {
			fmt.Println("Purging: ", room.Room_id)
			purge(baseURL, room.Room_id, client)
			i += 1
		}
	}

	fmt.Println("Purged ", i, " rooms")

	return err
}

func ls_room(baseURL string, client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", baseURL+"/_synapse/admin/v1/rooms", nil)
	if err != nil {
		return "", err
	}

	roomList, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer roomList.Body.Close()

	body, err := ioutil.ReadAll(roomList.Body)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	json.Indent(&out, body, "", "\t")

	return out.String(), err
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

	userList := flag.Bool("list", false, "List all users, requires no arguments")

	roomList := flag.Bool("room_list", false, "List all rooms, requires no arguments")

	deactivateTarget := flag.String("deactivate", "", "Deactivate an account, eg -deactivate @target:matrix.ais")
	resetTarget := flag.String("reset", "", "Reset users account with new password, eg -reset @target:matrix.ais")
	queryTarget := flag.String("query", "", "Queries a user and gets last ip, user agent, eg -query @target:matrix.ais")
	purgeTarget := flag.String("purge", "", "Purge a room from the database, typically so it can be reclaimed if everyone left, eg -purge !oqhoCmLzNgkVlLgxQp:matrix.ais, this can be found in the database of room_aliases")
	autoPurge := flag.Bool("autopurge", false, "Purge all rooms with 0 members joined to them")

	flag.Parse()

	u, err := url.Parse(*serverUrl)
	if err != nil {
		log.Fatal("Please enter valid URL")
	}

	if len(*deactivateTarget) == 0 && !*userList && len(*resetTarget) == 0 && len(*queryTarget) == 0 && len(*purgeTarget) == 0 && !*roomList && !*autoPurge {
		flag.PrintDefaults()
		log.Fatal("Please specify an option")

	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Admin username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Admin password: ")
	password := getSensitive()
	fmt.Print("\n")

	serverString := u.Scheme + "://" + u.Host

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

	if len(*deactivateTarget) != 0 {
		err = deactivate(serverString, *deactivateTarget, client)
	} else if *userList {
		err = ls(serverString, client)
	} else if len(*queryTarget) != 0 {
		err = query(serverString, *queryTarget, client)
	} else if len(*resetTarget) != 0 {
		fmt.Print("Enter new user password for ", *resetTarget, ": ")
		err = reset(serverString, *resetTarget, getSensitive(), client)
	} else if len(*purgeTarget) != 0 {
		err = purge(serverString, *purgeTarget, client)
	} else if *roomList {
		var rooms string
		rooms, err = ls_room(serverString, client)
		fmt.Println(rooms)
	} else if *autoPurge {
		err = autopurge(serverString, client)
	}

	if err != nil {
		log.Fatal(err)
	}

}
