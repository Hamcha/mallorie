package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func assert(err error) {
	if err != nil {
		log.Fatalf("FATAL ERROR: %s\r\n", err.Error())
	}
}

func main() {
	protocol := flag.String("proto", "tcp", "Protocol to use (tcp, udp)")
	target := flag.String("target", "", "Target to connect to. Format: <ip>:<port>")
	bind := flag.String("bind", "", "Address to bind the server to. Format [ip]:<port>")
	flag.Parse()

	if len(strings.TrimSpace(*target)) < 1 || len(strings.TrimSpace(*bind)) < 1 {
		log.Fatalln("Both -bind and -target arguments are required for mallorie to work. Please make sure both are specified and not empty.")
	}

	server, err := net.Listen(*protocol, *bind)
	assert(err)

	for {
		clientConn, err := server.Accept()
		if err != nil {
			log.Printf("Error while accepting connection: %s\r\n", err.Error())
			continue
		}

		serverConn, err := net.Dial(*protocol, *target)
		if err != nil {
			log.Printf("Cannot connect to target: %s\r\n", err.Error())
			clientConn.Close()
			continue
		}

		csReader, csWriter := io.Pipe()
		clientReader := io.TeeReader(clientConn, csWriter)
		cerrch := make(chan error)

		ssReader, ssWriter := io.Pipe()
		serverReader := io.TeeReader(serverConn, ssWriter)
		serrch := make(chan error)

		go sniff(csReader, "->")
		go sniff(ssReader, "<-")

		go func() {
			_, err := io.Copy(serverConn, clientReader)
			cerrch <- err
		}()
		go func() {
			_, err = io.Copy(clientConn, serverReader)
			serrch <- err
		}()

		select {
		case err = <-serrch:
			if err != nil {
				log.Printf("Client read error: %s\r\n", err.Error())
			}
		case err = <-cerrch:
			if err != nil {
				log.Printf("Server read error: %s\r\n", err.Error())
			}
		}
		serverConn.Close()
		clientConn.Close()
	}
}

func sniff(reader io.Reader, prefix string) {
	bufread := bufio.NewReader(reader)
	for {
		line, err := bufread.ReadString('\n')
		if err != nil {
			return
		}
		fmt.Printf("%s %s\r\n", prefix, strings.TrimSpace(line))
	}
}
