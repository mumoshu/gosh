/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a simple gRPC client that demonstrates how to use gRPC-Go libraries
// to perform unary, client streaming, server streaming and full duplex RPCs.
//
// It interacts with the route guide service whose definition can be found in routeguide/route_guide.proto.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mumoshu/gosh/data"
	pb "github.com/mumoshu/gosh/remote"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("server_addr", "localhost:10000", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name used to verify the hostname returned by the TLS handshake")
)

// runRemote receives a sequence of messages, while sending messages back to the client.
func runRemote(client pb.RemoteClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	log.Printf("%v.runRemote(_)", client)

	stream, err := client.ShellSession(ctx)
	if err != nil {
		log.Fatalf("%v.runRemote(_) = _, %v", client, err)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				log.Printf("Recv done")
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}
			log.Printf("Got message %s", in.Message)
		}
	}()

	reader := bufio.NewReader(os.Stdin)

	readCh := make(chan string)
	readErrCh := make(chan error)
	var cancelled bool

	go func() {
		defer close(readCh)
		defer close(readErrCh)

	FOR:
		for {
			done := make(chan struct{})

			var text string
			var err error

			go func() {
				os.Stdin.SetReadDeadline(time.Now().Add(1 * time.Second))
				text, err = reader.ReadString('\n')
				close(done)
			}()

		READ:
			for {
				var quit bool

				t := time.NewTimer(1 * time.Second)
				select {
				case <-ctx.Done():
					log.Printf("Stopping read")
					quit = true
				case <-t.C:
					log.Printf("Still waiting read")
					continue
				case <-done:
					log.Printf("Read returned")
				}

				if !t.Stop() {
					<-t.C
				}

				if quit {
					break FOR
				}

				break READ
			}

			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					if !cancelled {
						log.Printf("continuing")
						continue
					}
				}

				readErrCh <- err
				return
			}
			readCh <- text
		}
	}()

	errCh := make(chan error)
	func() {
		defer close(errCh)

	FOR:
		for {
			select {
			case text, ok := <-readCh:
				if text != "" {
					text = strings.Replace(text, "\n", "", -1)
					note := &pb.Message{Message: text}

					if err := stream.Send(note); err != nil {
						errCh <- err
						break FOR
					}
				}

				if !ok {
					readCh = nil
				}
			case err, ok := <-readErrCh:
				if err != nil {
					errCh <- err
					break FOR
				}
				if !ok {
					readErrCh = nil
				}
			case <-waitc:
				waitc = nil
				break FOR
			}
		}
	}()

	log.Printf("Closing send")
	stream.CloseSend()

	cancelled = true

	log.Printf("Cancelling")

	cancel()

	if err := <-errCh; err != nil {

		log.Fatalf("Failed to send a note: %v", err)
	}

	log.Printf("Ending")
}

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	if *tls {
		if *caFile == "" {
			*caFile = data.Path("x509/ca_cert.pem")
		}
		creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
		if err != nil {
			log.Fatalf("Failed to create TLS credentials %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewRemoteClient(conn)

	runRemote(client)
}
