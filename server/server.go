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

// Package main implements a simple gRPC server that demonstrates how to use gRPC-Go libraries
// to perform unary, client streaming, server streaming and full duplex RPCs.
//
// It implements the route guide service whose definition can be found in routeguide/route_guide.proto.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/mumoshu/gosh/data"
	"google.golang.org/grpc/credentials"

	pb "github.com/mumoshu/gosh/remote"
)

var (
	tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile = flag.String("cert_file", "", "The TLS cert file")
	keyFile  = flag.String("key_file", "", "The TLS key file")
	port     = flag.Int("port", 10000, "The server port")
)

type server struct {
	pb.UnimplementedRemoteServer

	mu       sync.Mutex
	messages map[string][]*pb.Message
}

// RouteChat receives a stream of message/location pairs, and responds with a stream of all
// previous messages at each of those locations.
func (s *server) ShellSession(stream pb.Remote_ShellSessionServer) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir(filepath.Join(wd, "sessions"), "xx")
	if err != nil {
		return err
	}

	recvErrCh := make(chan error)
	sendOutErrCh := make(chan error)
	sendErrErrCh := make(chan error)
	runErrCh := make(chan error)

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, "bash", "-e")
	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW
	cmd.Dir = tmpDir

	go func() {
		defer close(recvErrCh)
		defer stdinW.Close()

		log.Printf("Receiving")

	FOR:
		for {
			done := make(chan struct{})

			var in *pb.Message
			go func() {
				in, err = stream.Recv()
				close(done)
			}()

		RECV:
			for {
				var quit bool

				t := time.NewTimer(500 * time.Millisecond)
				select {
				case <-ctx.Done():
					log.Printf("Stopping recv")
					quit = true
				case <-t.C:
					log.Printf("Retrying recv")
					// This seems to be needed to wake/unblock cmd.Wait()
					// _, err = fmt.Fprintf(stdinW, "")
					cmd.Process.Signal(syscall.SIGCONT)
					continue
				case <-done:
					log.Printf("Recv returned")
				}

				if !t.Stop() {
					<-t.C
				}

				if quit {
					break FOR
				}

				break RECV
			}

			if in != nil {
				log.Printf("Received %s", in.Message)
			} else {
				log.Printf("Received err %v", err)
			}

			if err == io.EOF {
				_, err = fmt.Fprintf(stdinW, "echo existing....; exit\n")
			}

			if err != nil {
				recvErrCh <- err
				return
			}

			if in == nil {
				return
			}

			_, err = fmt.Fprintf(stdinW, in.Message+"\n")
			if err != nil {
				recvErrCh <- err
				return
			}
		}
	}()

	go func() {
		defer close(sendOutErrCh)

		log.Printf("Sending stdout")

		s := bufio.NewScanner(stdoutR)

		for s.Scan() {
			log.Printf("out: %s", s.Text())
			if err := stream.Send(&pb.Message{Message: s.Text()}); err != nil {
				log.Printf("Error sending stdout: %v", err)

				sendOutErrCh <- err
				return
			}
		}
	}()

	go func() {
		defer close(sendErrErrCh)

		log.Printf("Sending stderr")

		s := bufio.NewScanner(stderrR)

		for s.Scan() {
			log.Printf("err: %s", s.Text())
			if err := stream.Send(&pb.Message{Message: s.Text()}); err != nil {
				log.Printf("Error sending stderr: %v", err)
				sendErrErrCh <- err
				return
			}
		}
	}()

	log.Printf("Starting command")

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		defer close(runErrCh)

		waitErr := cmd.Wait()

		log.Printf("Command finished: %+v", waitErr)

		if waitErr != nil {
			runErrCh <- err
		}
	}()

	var errs []error

	done := ctx.Done()

	type state struct {
		done bool
		err  error
	}

	type status struct {
		recv, out, err, run state
	}

	var st status

	f := func() bool {
		return st.err.done && st.out.done && st.run.done
	}

FOR:
	for {
		select {
		case err, ok := <-recvErrCh:
			if !st.recv.done {
				st.recv.done = true
				log.Printf("recvErr: %v", err)
				fmt.Fprintf(stdinW, "echo force exiting due to recv err...; exit\n")
				st.recv.err = err
			}

			if !ok {
				recvErrCh = nil
				log.Printf("recvErr is already closed")
			}

			if f() {
				break FOR
			}
		case err, ok := <-sendErrErrCh:
			if !st.err.done {
				st.err.done = true
				log.Printf("sendErrErr: %v", err)
				cancel()
				st.err.err = err
			}

			if !ok {
				sendErrErrCh = nil
				log.Printf("sendErrErr is already closed")
			}

			if f() {
				break FOR
			}
		case err, ok := <-sendOutErrCh:
			if !st.out.done {
				st.out.done = true
				log.Printf("sendOutErr: %v", err)
				cancel()
				st.out.err = err
			}

			if !ok {
				sendOutErrCh = nil
				log.Printf("sendOutErr is already closed")

			}

			if f() {
				break FOR
			}
		case err, ok := <-runErrCh:
			if !st.run.done {
				st.run.done = true
				st.run.err = err
				log.Printf("runErr: %v", err)
				cancel()
			}

			if !ok {
				runErrCh = nil
				log.Printf("runErr is already closed")
			}

			if f() {
				break FOR
			}
		case _, ok := <-done:
			if !ok {
				done = nil
			}

			log.Printf("cancelled")
			stdoutW.Close()
			stderrW.Close()
		}
	}

	for _, e := range errs {
		if e != nil {
			return err
		}
	}

	log.Printf("Closing bi-directional stream")

	return nil
}

func newServer() *server {
	s := &server{messages: make(map[string][]*pb.Message)}
	return s
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = data.Path("x509/server_cert.pem")
		}
		if *keyFile == "" {
			*keyFile = data.Path("x509/server_key.pem")
		}
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRemoteServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}
