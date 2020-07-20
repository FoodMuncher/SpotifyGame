package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
)

const (
	port  = 8080
	scope = "user-read-private user-read-email"
)

var (
	redirectURI = "http://localhost:" + strconv.Itoa(port) + "/callback"
	config      struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	authEndpoint string
)

type auth struct {
	Endpoint string
}

// Token Struct for Decoding Access Token Call Response
type token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func main() {
	setConfig()

	fmt.Println("Spotify API")
	fmt.Println("-------------------")
	fmt.Println("Running on: http://localhost:8080/")

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/login", logInHandler)
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/redirect", redirectHandler)
	http.HandleFunc("/form", formHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(writer http.ResponseWriter, request *http.Request) {
	fmt.Println("Id:", config.ClientID)
	fmt.Println("secret:", config.ClientSecret)
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

		fmt.Fprint(writer, string(userResponseBody))
	} else {
		http.Redirect(writer, request, "http://localhost:8080/login", 307)
	}
}

func logInHandler(writer http.ResponseWriter, request *http.Request) {
	template := template.Must(template.ParseFiles("./templates/login.html"))
	err := template.Execute(writer, auth{authEndpoint})
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

	var token token
	err = json.Unmarshal(responseBody, &token)
	checkError(err)

	if token.AccessToken != "" {
		accessCookie := http.Cookie{
			Name:  "access_token",
			Value: token.AccessToken,
		}
		http.SetCookie(writer, &accessCookie)

		refreshCookie := http.Cookie{
			Name:  "refresh_token",
			Value: token.RefreshToken,
		}
		http.SetCookie(writer, &refreshCookie)

		http.Redirect(writer, request, "http://localhost:8080/redirect", 307)
	}
}

func formHandler(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	fmt.Println(request.Form["session_id"])
}

func redirectHandler(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie("access_token")
	checkError(err)

	fmt.Println(cookie)
	fmt.Fprintf(writer, "<h1>Yo!</h1>")
}

func setConfig() {
	configFile, err := os.Open("config.json")
	checkError(err)

	defer configFile.Close()

	configBytes, err := ioutil.ReadAll(configFile)
	checkError(err)

	json.Unmarshal(configBytes, &config)

	fmt.Println("https://accounts.spotify.com/authorize/?response_type=code&client_id=" + config.ClientID + "&redirect_uri=" + redirectURI + "&scope=" + scope)
	fmt.Println(config.ClientID)

	authEndpoint = "https://accounts.spotify.com/authorize/?response_type=code&client_id=" + config.ClientID + "&redirect_uri=" + redirectURI + "&scope=" + scope
}

func checkError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
