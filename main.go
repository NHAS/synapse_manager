// synapse_manage project main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

var protected = map[string]bool{
	"#pentest:matrix.ais":      true,
	"#pentest-help:matrix.ais": true,
	"#scoping:matrix.ais":      true,
	"#noot:matrix.ais":         true,
	"#oscx:matrix.ais":         true,
	"#poltics:matrix.ais":      true,
	"#reporting:matrix.ais":    true,
	"#sot:matrix.ais":          true,
	"#vso:matrix.ais":          true,
	"#wellington:matrix.ais":   true,
	"#phishing:matrix.ais":     true,
	"#aws:matrix.ais":          true,
}

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

type Room struct {
	Room_id         string
	Name            string
	Canonical_alias string
	Creator         string `json:"creator"`
	Encryption      string `json:"encryption"`
	JoinedMembers   int    `json:"joined_local_members"`
}

type RoomsList struct {
	Offset      int
	Total_rooms int
	Rooms       []Room
}

//Apply auth token to each request we have to make
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

	if len(authResp.AccessToken) == 0 {
		return "", errors.New("Access token was empty for some reason. " + string(body))
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
	req, err := http.NewRequest("GET", baseURL+"/_synapse/admin/v2/users/"+who, nil) // This call is set to be depricated, however the stated replacement doesnt work as of synapse 1.9.0
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

	userAttr := struct {
		Password    string `json:"password"`
		Deactivated bool   `json:"deactivated"`
	}{
		Password:    pass,
		Deactivated: false,
	}

	b, _ := json.Marshal(&userAttr)

	req, err := http.NewRequest("PUT", baseURL+"/_synapse/admin/v2/users/"+who, bytes.NewReader(b))
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return err
	}

	resetResponse, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resetResponse.Body.Close()

	b, err = ioutil.ReadAll(resetResponse.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(b))
	return nil
}

func delete(baseURL, room string, client *http.Client) error {

	req, err := http.NewRequest("POST", baseURL+"/_synapse/admin/v1/rooms/"+room+"/delete", bytes.NewBuffer([]byte("{}")))
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

	if bytes.Contains(body, []byte("errcode")) {
		return errors.New("Unable to delete room: " + string(body))
	}

	fmt.Println(string(body))
	return nil
}

func autodelete(baseURL string, client *http.Client) error {

	list, err := ls_room(baseURL, client)

	var listRooms RoomsList
	err = json.Unmarshal([]byte(list), &listRooms)
	if err != nil {
		return err
	}

	type roomDetails struct {
		name   string
		roomID string
	}

	var roomsToDelete []roomDetails

	i := 0
	for _, room := range listRooms.Rooms {

		if room.JoinedMembers == 0 { // Currently you can only destroy rooms with 0 members
			if _, ok := protected[strings.TrimSpace(room.Canonical_alias)]; ok {
				fmt.Print("\nLooks like autodelete is trying to delete protected room ", room.Canonical_alias, "(", room.Room_id, "). Unable to delete protected rooms as failsafe")
				return nil
			}
			roomsToDelete = append(roomsToDelete, roomDetails{room.Name, room.Room_id})

			i++
		}
	}

	fmt.Println(i, " rooms to delete")
	for _, m := range roomsToDelete {
		fmt.Println("\t", m.name, ":", m.roomID)
	}
	fmt.Print("Continue? (N/y) ")

	var response string
	_, err = fmt.Scanln(&response)
	response = strings.TrimSpace(response)
	if response != "y" && response != "Y" {
		return nil
	}

	for _, m := range roomsToDelete {
		fmt.Println("Deleting ", m.roomID)
		err := delete(baseURL, m.roomID, client)
		if err != nil {
			log.Println("\t", err)
		}
	}

	fmt.Println("Deleted ", i, " rooms")

	return nil
}

func ls_room(baseURL string, client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", baseURL+"/_synapse/admin/v1/rooms", nil)
	if err != nil {
		return "", err
	}

	listRooms, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer listRooms.Body.Close()

	body, err := ioutil.ReadAll(listRooms.Body)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	json.Indent(&out, body, "", "\t") // Format json so a human can read it

	return out.String(), err
}

func checkEncrypt(baseURL string, client *http.Client) error {
	rooms, err := ls_room(baseURL, client)
	if err != nil {
		return err
	}

	var rl RoomsList
	err = json.Unmarshal([]byte(rooms), &rl)
	if err != nil {
		return err
	}

	for _, v := range rl.Rooms {
		if !strings.Contains(v.Encryption, "olm") {

			log.Println("Room Id: ", v.Room_id)
			log.Println("Name: ", v.Name)
			log.Println("Creator: ", v.Creator)
			log.Println("Encryption: ", v.Encryption)

		}
	}
	return nil
}

func forceJoin(baseURL, userName, room string, client *http.Client) error {

	user := struct {
		UserID string `json:"user_id"`
	}{
		UserID: userName,
	}

	b, err := json.Marshal(&user)
	if err != nil {
		return err
	}

	listRooms, err := client.Post(baseURL+"/_synapse/admin/v1/join/"+room, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer listRooms.Body.Close()

	body, err := ioutil.ReadAll(listRooms.Body)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	json.Indent(&out, body, "", "\t") // Format json so a human can read it

	fmt.Println(out.String())

	return err
}

func deleteOldContent(baseURL string, client *http.Client) error {

	clearContent, err := client.Post(fmt.Sprintf("%s/_synapse/admin/v1/media/matrix.ais/delete?before_ts=%d", baseURL, time.Now().UnixNano()-604800), "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	defer clearContent.Body.Close()

	body, err := ioutil.ReadAll(clearContent.Body)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	json.Indent(&out, body, "", "\t") // Format json so a human can read it

	fmt.Println(out.String())

	return err
}

func getSensitive() string {
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin)) // Turns off stdin echo
	if err != nil {
		log.Fatal(err)
	}
	password := string(bytePassword)

	return strings.TrimSpace(password)
}

func main() {

	serverUrl := flag.String("url", "http://localhost:8008", "The URL that points towards the matrix server")

	userList := flag.Bool("ls_users", false, "List all users, requires no arguments")

	listRooms := flag.Bool("ls_rooms", false, "List all rooms, requires no arguments")
	autoDelete := flag.Bool("auto_delete", false, "Delete all rooms with 0 members joined to them")
	checkEncryption := flag.Bool("check_encryption", false, "Check encryption is enabled on all rooms, prints any room without encryption")

	deactivateTarget := flag.String("deactivate", "", "Deactivate an account, eg -deactivate @target:matrix.ais")
	resetTarget := flag.String("reset", "", "Reset users account with new password, eg -reset @target:matrix.ais")
	queryTarget := flag.String("query", "", "Queries a user and gets last ip, user agent, eg -query @target:matrix.ais")
	deleteTarget := flag.String("delete", "", "Delete a room from the database, typically so it can be reclaimed if everyone left, eg -delete !oqhoCmLzNgkVlLgxQp:matrix.ais, this can be found in the database of room_aliases")
	forceJoinTarget := flag.String("join", "", "Target join to a room, e.g -forceJoinTarget @target:matrix.ais")

	deleteContent := flag.Bool("delete_old_content", false, "Delete all local content on the server older than a week")

	flag.Parse()

	u, err := url.Parse(*serverUrl)
	if err != nil {
		log.Fatal("Please enter valid URL")
	}

	if !*deleteContent && len(*forceJoinTarget) == 0 && len(*deactivateTarget) == 0 && !*userList && len(*resetTarget) == 0 && len(*queryTarget) == 0 && len(*deleteTarget) == 0 && !*checkEncryption && !*listRooms && !*autoDelete {
		flag.PrintDefaults()
		log.Fatal("Please specify an option")

	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Fprint(os.Stderr, "Synapse admin username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Fprint(os.Stderr, "Synapse admin password: ")
	password := getSensitive() // Turn off echoing of stdin
	fmt.Print("\n")

	serverString := u.Scheme + "://" + u.Host

	token, err := login(serverString, username, password)
	if err != nil {
		log.Fatal(err)
	}

	// Once we have the auth token, apply it to every API call we make, and destroy it after
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
	} else if len(*deleteTarget) != 0 {
		err = delete(serverString, *deleteTarget, client)
	} else if *listRooms {
		var rooms string
		rooms, err = ls_room(serverString, client)
		fmt.Println(rooms)
	} else if *autoDelete {
		err = autodelete(serverString, client)
	} else if *checkEncryption {
		err = checkEncrypt(serverString, client)
	} else if len(*forceJoinTarget) != 0 {
		fmt.Fprint(os.Stderr, "Room to join: ")
		room, _ := reader.ReadString('\n')
		room = strings.TrimSpace(room)

		err = forceJoin(serverString, *forceJoinTarget, room, client)
	} else if *deleteContent {
		err = deleteOldContent(serverString, client)
	}

	if err != nil {
		log.Fatal(err)
	}

}
