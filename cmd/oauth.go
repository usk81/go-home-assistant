package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/oauth2"
)

const oauthTokenFilename = "oauthTokenCache"

var (
	oauthToken       *oauth2.Token
	gcp              *gcpAuthWrapper
	oauthSrv         *http.Server
	oauthRedirectURL = "http://localhost:8080"
)

type JSONToken struct {
	Installed struct {
		ClientID                string   `json:"client_id"`
		ProjectID               string   `json:"project_id"`
		AuthURI                 string   `json:"auth_uri"`
		TokenURI                string   `json:"token_uri"`
		AuthProviderX509CertURL string   `json:"auth_provider_x509_cert_url"`
		ClientSecret            string   `json:"client_secret"`
		RedirectUris            []string `json:"redirect_uris"`
	} `json:"installed"`
}

type gcpAuthWrapper struct {
	Conf      *oauth2.Config
	TokenPath string
}

func (w *gcpAuthWrapper) Start() {
	f, err := os.Open(creds)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var token JSONToken
	if err = json.NewDecoder(f).Decode(&token); err != nil {
		log.Println("failed to decode json token", err)
		panic(err)
	}

	w.Conf = &oauth2.Config{
		ClientID:     token.Installed.ClientID,
		ClientSecret: token.Installed.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/assistant-sdk-prototype"},
		RedirectURL:  oauthRedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://accounts.google.com/o/oauth2/token",
		},
	}

	if logout {
		fmt.Println("Deleting potential oauth cache")
		os.Remove(w.TokenPath)
	}

	// check if we have an oauth file on disk
	if hasCachedOauth(w.TokenPath) {
		err = loadTokenSource(w.TokenPath)
		if err == nil {
			fmt.Println("Launching the Google Assistant using cached credentials")
			return
		}
		fmt.Println("Failed to load the token source", err)
		fmt.Println("Continuing program without cached credentials")
	}

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	url := w.Conf.AuthCodeURL("state", oauth2.AccessTypeOffline)

	if runtime.GOOS != "darwin" {
		fmt.Printf("Copy and paste the following url into your browser to authenticate:\n%s\n", url)
	} else {
		cmd := exec.Command("open", url)
		cmd.Run()
	}
	// if we are using the builtin auth server locally
	if remote {
		// remote access
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Enter the auth code followed by enter")
		permissionCode, _ := reader.ReadString('\n')
		setTokenSource(w.TokenPath, permissionCode)
	} else {
		// Start the server to receive the code
		oauthSrv = &http.Server{Addr: ":8080", Handler: http.DefaultServeMux}
		http.HandleFunc("/", w.oauthHandler)
		err = oauthSrv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}
	fmt.Println("Launching the Google Assistant")
}

func (w *gcpAuthWrapper) oauthHandler(wr http.ResponseWriter, r *http.Request) {
	permissionCode := r.URL.Query().Get("code")
	// TODO: check the status code
	wr.Write([]byte(fmt.Sprintf("<h1>Your code is: %s</h1>", permissionCode)))
	setTokenSource(w.TokenPath, permissionCode)
	// kill the http server
	oauthSrv.Shutdown(context.Background())
}

func hasCachedOauth(tokenPath string) bool {
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return false
	}
	return true
}

func setTokenSource(tokenPath, permissionCode string) {
	var err error
	ctx := context.Background()
	oauthToken, err = gcp.Conf.Exchange(ctx, permissionCode)
	if err != nil {
		fmt.Println("failed to retrieve the oauth2 token")
		log.Fatal(err)
	}
	fmt.Println(oauthToken)
	of, err := os.Create(tokenPath)
	if err != nil {
		panic(err)
	}
	defer of.Close()
	if err = json.NewEncoder(of).Encode(oauthToken); err != nil {
		log.Println("Something went wrong when storing the token source", err)
		panic(err)
	}
}

func loadTokenSource(tokenPath string) error {
	f, err := os.Open(tokenPath)
	if err != nil {
		return fmt.Errorf("failed to load the token source (deleted from disk) %v", err)
	}
	defer f.Close()
	var token oauth2.Token
	if err = json.NewDecoder(f).Decode(&token); err != nil {
		return err
	}
	oauthToken = &token
	return nil
}
