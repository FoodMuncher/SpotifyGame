package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
)

const (
	port         = 8080
	clientID     = "4ccc8676aaf54c94a6400ce027c1c93e"
	clientSecret = "0fe59c37743341909e72e45664ea5f73"
	scope        = "user-read-currently-playing"
)

var (
	redirectURI  = "http://localhost:" + strconv.Itoa(port) + "/callback"
	authEndpoint = "https://accounts.spotify.com/authorize/?response_type=code&client_id=" + clientID + "&redirect_uri=" + redirectURI + "&scope=" + scope
)

type auth struct {
	Endpoint string
}

// Token Struct for Decoding Token Call Response
type token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type currentTrack struct {
	Item struct {
		Album struct {
			Images []struct {
				Height int    `json:"height"`
				Width  int    `json:"width"`
				URL    string `json:"url"`
			} `json:"images"`
			Name string `json:"name"`
		} `json:"album"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
		Name string `json:"name"`
	} `json:"item"`
}

func main() {
	fmt.Println("Spotify API")
	fmt.Println("-------------------")
	fmt.Println("Running on: http://localhost:8080/")

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/callback", callbackHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(writer http.ResponseWriter, reader *http.Request) {

	auth := auth{authEndpoint}

	path := "./templates/home.html"

	template := template.Must(template.ParseFiles(path))
	err := template.Execute(writer, auth)
	checkError(err)
}

func callbackHandler(writer http.ResponseWriter, reader *http.Request) {
	authCode := reader.URL.Query().Get("code")
	client := http.Client{}
	requestBody := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {authCode},
		"redirect_uri": {redirectURI},
	}.Encode()

	tokenURI := "https://accounts.spotify.com/api/token"
	encodedClientString := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))

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
		request1, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/currently-playing", nil)
		request1.Header.Set("Authorization", "Bearer "+token.AccessToken)
		checkError(err)

		response1, err := client.Do(request1)
		checkError(err)

		defer response1.Body.Close()

		responseBody1, err := ioutil.ReadAll(response1.Body)
		checkError(err)

		var currentTrack currentTrack

		err = json.Unmarshal(responseBody1, &currentTrack)
		checkError(err)

		presentInformation(writer, currentTrack)
	}
}

func presentInformation(writer http.ResponseWriter, currentTrack currentTrack) {
	albumArtURL := "src=" + currentTrack.Item.Album.Images[0].URL
	albumArtHeight := " height=" + strconv.Itoa(currentTrack.Item.Album.Images[0].Height)
	albumArtWidth := " width=" + strconv.Itoa(currentTrack.Item.Album.Images[0].Width)

	title := currentTrack.Item.Name
	artist := currentTrack.Item.Artists[0].Name
	album := currentTrack.Item.Album.Name

	// TODO: template this....
	fmt.Fprintf(writer, ` 
		<!DOCTYPE html>
		<html>
			<body>
			<button onclick="window.location.href='http://localhost:8080/';">
    			Home
  			</button>
			<h2>Playing "`+title+`" By "`+artist+`" On "`+album+`"</h2>
				<img `+albumArtURL+albumArtHeight+albumArtWidth+`></img>
			</body>
		</html>
		`)
}

func checkError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
