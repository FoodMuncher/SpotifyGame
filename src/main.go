package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	store "./store"
)

// =====================================
// Global Constants
// =====================================

const (
	port            = 8080
	scope           = "user-read-private user-read-email"
	charset         = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	sessionIDLength = 15
)

const (
	add    = "ADD"
	remove = "REMOVE"
)

// =====================================
// Global Variables
// =====================================

var (
	seededRand  *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	redirectURI            = "http://localhost:" + strconv.Itoa(port) + "/callback"
	config      struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	authEndpoint string
)

var (
	sessionStore = store.NewStore()
	indexStore   = store.NewStore()
)

// =====================================
// Template Structs
// =====================================

type loginPage struct {
	AuthEndpoint string
}

type homePage struct {
	ProfilePicture string
	SessionID      string
}

type hostPage struct {
	ProfilePicture string
	SessionID      string
	List           []string
}

// =====================================
// JSON Structs
// =====================================

// Struct to hold all information from user information call.
type userInformationJSON struct {
	Images []struct {
		URL string `json:"url"`
	} `json:"images"`
	URI   string `json:"uri"`
	Error spotifyErrorJSON
}

type spotifyErrorJSON struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Token Struct for Decoding Access Token Call Response
type tokenJSON struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// =====================================
// Functions
// =====================================

func main() {
	setConfig()

	fmt.Println("Spotify API")
	fmt.Println("-------------------")
	fmt.Println("Running on: http://localhost:8080/")

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/login", logInHandler)
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/host", hostHandler)
	http.HandleFunc("/form", formHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(writer http.ResponseWriter, request *http.Request) {
	accessCookie, err := request.Cookie("access_token")
	if err == nil {
		userRequest, err := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
		userRequest.Header.Set("Authorization", "Bearer "+accessCookie.Value)
		checkError(err)

		client := &http.Client{}
		userResponse, err := client.Do(userRequest)
		checkError(err)

		defer userResponse.Body.Close()

		userResponseBody, err := ioutil.ReadAll(userResponse.Body)
		checkError(err)

		var user userInformationJSON

		checkError(json.Unmarshal(userResponseBody, &user))

		if user.Error == (spotifyErrorJSON{}) {
			// TODO: check cookies first then if they don't exist, fetch them
			setCookie(writer, "profile_picture", user.Images[0].URL)
			setCookie(writer, "URI", user.URI)

			template := template.Must(template.ParseFiles("./templates/home.html"))
			checkError(template.Execute(writer, homePage{user.Images[0].URL, createSessionID()}))
		} else if user.Error.Status == 401 {
			// Refresh Token
			fmt.Println("401 son")
		} else {
			http.Redirect(writer, request, "http://localhost:8080/login", 307)
		}
	} else {
		http.Redirect(writer, request, "http://localhost:8080/login", 307)
	}
}

func logInHandler(writer http.ResponseWriter, request *http.Request) {
	template := template.Must(template.ParseFiles("./templates/login.html"))
	err := template.Execute(writer, loginPage{authEndpoint})
	checkError(err)
}

func callbackHandler(writer http.ResponseWriter, request *http.Request) {
	authCode := request.URL.Query().Get("code")
	client := http.Client{}
	requestBody := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {authCode},
		"redirect_uri": {redirectURI},
	}.Encode()

	tokenURI := "https://accounts.spotify.com/api/token"
	encodedClientString := base64.StdEncoding.EncodeToString([]byte(config.ClientID + ":" + config.ClientSecret))

	request, err := http.NewRequest("POST", tokenURI, strings.NewReader(requestBody))
	request.Header.Set("Authorization", "Basic "+encodedClientString)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	checkError(err)

	response, err := client.Do(request)
	checkError(err)

	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	checkError(err)

	var token tokenJSON
	err = json.Unmarshal(responseBody, &token)
	checkError(err)

	if token.AccessToken != "" {
		setCookie(writer, "access_token", token.AccessToken)
		setCookie(writer, "refresh_token", token.RefreshToken)

		http.Redirect(writer, request, "http://localhost:8080/", 307)
	}
}

// HostHandler - The page a user goes to after they log in and host a session.
// TODO - have intermidiate endpoint that stores info then redirects to a different page. (Prevents from storeing same values multiple times...)
func hostHandler(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	sessionID := request.Form.Get("session_id")

	URI, err := request.Cookie("URI")
	checkError(err)

	sessionStore.Add(sessionID, URI.Value)
	indexStore.Add(URI.Value, sessionID)

	profilePicture, err := request.Cookie("profile_picture")

	template := template.Must(template.ParseFiles("./templates/host.html"))
	checkError(template.Execute(writer, hostPage{profilePicture.Value, sessionID, []string{"Song 1", "Song 2", "Song 3"}}))
}

func formHandler(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	fmt.Println(request.Form["session_id"])
}

// FaviconHandler - Handles serving of favicon.icof
func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/spotify.ico")
}

func setConfig() {
	configFile, err := os.Open("config.json")
	checkError(err)

	defer configFile.Close()

	configBytes, err := ioutil.ReadAll(configFile)
	checkError(err)

	json.Unmarshal(configBytes, &config)

	authEndpoint = "https://accounts.spotify.com/authorize/?response_type=code&client_id=" + config.ClientID + "&redirect_uri=" + redirectURI + "&scope=" + scope
}

func createSessionID() string {
	b := make([]byte, sessionIDLength)
	uniqueID := true

	for uniqueID {
		for i := range b {
			b[i] = charset[seededRand.Intn(len(charset))]
		}

		uniqueID = sessionStore.Read(string(b)) != nil
	}
	return string(b)
}

func setCookie(writer http.ResponseWriter, name, value string) {
	http.SetCookie(writer, &http.Cookie{
		Name:  name,
		Value: value,
	})
}

func checkError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
