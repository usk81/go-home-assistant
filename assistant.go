package assistant

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gordonklaus/portaudio"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
	embedded "google.golang.org/genproto/googleapis/assistant/embedded/v1alpha2"
	"google.golang.org/grpc"
)

const (
	// APIEndpoint is Google Assistant API endpoint
	APIEndpoint = "embeddedassistant.googleapis.com:443"

	// ScopeAssistantSDK is The API scope for Google Assistant
	ScopeAssistantSDK = "https://www.googleapis.com/auth/assistant-sdk-prototype"
)

type Client struct {
	Context context.Context
	Config  embedded.AssistConfig
	Token   oauth2.TokenSource
	Timeout time.Duration
}

type Request struct {
	Context context.Context
	Config  embedded.AssistConfig
	Token   oauth2.TokenSource
	Timeout time.Duration
}

func New(r Request) *Client {
	return &Client{
		Context: r.Context,
		Config:  r.Config,
		Token:   r.Token,
		Timeout: r.Timeout,
	}
}

func (c *Client) Call(tq string) (err error) {
	cf := c.Config
	if tq != "" {
		cf.Type = &embedded.AssistConfig_TextQuery{
			TextQuery: tq,
		}
	}
	call(c.Context, c.Timeout, c.Token, &cf)
	return nil
}

func newConn(ctx context.Context, ts oauth2.TokenSource) (conn *grpc.ClientConn, err error) {
	return transport.DialGRPC(ctx,
		option.WithTokenSource(ts),
		option.WithEndpoint(APIEndpoint),
		option.WithScopes(ScopeAssistantSDK),
	)
}

func call(cx context.Context, timeout time.Duration, ts oauth2.TokenSource, conf *embedded.AssistConfig) (err error) {
	ctx, canceler := context.WithTimeout(cx, timeout)

	stop := func() {
		ctx.Done()
		canceler()
	}
	defer stop()

	conn, err := newConn(ctx, ts)
	if err != nil {
		log.Println("failed to acquire connection", err)
		return
	}
	defer conn.Close()

	assistant := embedded.NewEmbeddedAssistantClient(conn)

	a, err := assistant.Assist(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Google Assistant: %w", err)
	}

	// listening in the background
	go func() {
		err = a.Send(&embedded.AssistRequest{
			Type: &embedded.AssistRequest_Config{
				Config: conf,
			},
		})

		if err != nil {
			log.Printf("Could not send audio: %v", err)
		}
		a.CloseSend()
	}()

	// audio out
	bufOut := make([]int16, 799)

	// var bufWriter bytes.Buffer
	streamOut, err := portaudio.OpenDefaultStream(0, 1, 16000, len(bufOut), &bufOut)
	defer func() {
		if err := streamOut.Close(); err != nil {
			// log.Println("failed to close the stream", err)
		}
		// log.Println("stream closed")
	}()
	if err = streamOut.Start(); err != nil {
		log.Println("failed to start audio out")
		panic(err)
	}

	// log.Println("Listening")
	// waiting for google assistant response
	for {
		resp, err := a.Recv()
		if err == io.EOF {
			// log.Println("we are done!!!!")
			break
		}
		if err != nil {
			log.Fatalf("Cannot get a response from the assistant: %v", err)
			continue
		}

		if resp.GetEventType() == embedded.AssistResponse_END_OF_UTTERANCE {
			log.Println("Google said you are done, are you?!")
		}
		audioOut := resp.GetAudioOut()
		if audioOut != nil {
			// log.Printf("audio out from the assistant (%d bytes)\n", len(audioOut.AudioData))

			signal := bytes.NewBuffer(audioOut.AudioData)
			var err error
			for err == nil {
				err = binary.Read(signal, binary.LittleEndian, bufOut)
				if err != nil {
					break
				}

				if portErr := streamOut.Write(); portErr != nil {
					log.Println(fmt.Errorf("Failed to write to audio out : %w", portErr))
				}
			}
		}
	}
	return nil
}
