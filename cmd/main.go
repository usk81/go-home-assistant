package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"flag"

	"github.com/gordonklaus/portaudio"
	"github.com/mitchellh/go-homedir"
	assistant "github.com/usk81/go-home-assistant"
)

const (
	// APIEndpoint is Google Assistant API endpoint
	APIEndpoint = "embeddedassistant.googleapis.com:443"

	// ScopeAssistantSDK is The API scope for Google Assistant
	ScopeAssistantSDK = "https://www.googleapis.com/auth/assistant-sdk-prototype"
)

var (
	// Debug allows the caller to see more debug print messages.
	Debug bool

	creds  string
	cache  string
	lang   string
	logout bool
	remote bool
)

func main() {
	flag.StringVar(&creds, "creds", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"), "path to the credentials file")
	flag.StringVar(&cache, "cache", mustCredentialFilePath(), "path to a google access token source")
	flag.StringVar(&lang, "lang", "en-US", "language code")
	flag.BoolVar(&logout, "logout", false, "should the current user be logged out")
	flag.BoolVar(&remote, "remote", false, "is the machine running the program accessed remotely (via SSH for instance)")
	flag.Parse()
	if creds == "" {
		fmt.Println("you need to provide a path to your credentials or set GOOGLE_APPLICATION_CREDENTIALS")
		os.Exit(1)
	}
	if cache == "" {
		cache = oauthTokenFilename
	}
	if remote {
		oauthRedirectURL = "urn:ietf:wg:oauth:2.0:oob"
	}

	args := flag.Args()
	var query string
	if len(args) > 0 {
		query = args[0]
	}

	// connect to the audio drivers
	portaudio.Initialize()
	defer portaudio.Terminate()

	gcp = &gcpAuthWrapper{
		TokenPath: cache,
	}
	gcp.Start()

	config := assistant.GetDefaultConfig()

	if lang != "" {
		config.DialogStateIn.LanguageCode = lang
	}

	ctx := context.Background()
	timeout := 240 * time.Second
	ts := gcp.Conf.TokenSource(ctx, oauthToken)

	cli := assistant.New(assistant.Request{
		Context: ctx,
		Config:  config,
		Token:   ts,
		Timeout: timeout,
	})

	if err := cli.Call(query); err != nil {
		fmt.Printf("%s", err.Error())
	}
}

func credentialFilePath() (fp string, err error) {
	dir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, oauthTokenFilename), nil
}

func mustCredentialFilePath() (fp string) {
	fp, err := credentialFilePath()
	if err != nil {
		return ""
	}
	return fp
}
