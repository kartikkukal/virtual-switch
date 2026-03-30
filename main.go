package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/google/uuid"
)

type Packet struct {
	To   string
	From string

	Head string

	Data any
}

const (
	ConnectedNode = "connected_node"
	RemovedNode   = "removed_node"

	GeneralTextMessage = "general_text_message"
)

var (
	hasDefaultRoute = false
	defaultRoute    chan Packet

	address = uuid.New().String()

	sockets map[string]chan Packet
	routing map[string]string

	ErrUnexpectedType = errors.New("packet contains unexpected type")
)

func IncomingPacketHandler(packet Packet) error {

	logger := slog.With("method", "IncomingPacketHandler")

	if packet.Head == GeneralTextMessage {

		data, ok := packet.Data.(string)
		if !ok {
			return ErrUnexpectedType
		}

		fmt.Print("\nIncoming message\n>", data, "\n")
		return nil
	}

	return nil
}

func NetworkPacketHandler(packet Packet, socket string) error {

	logger := slog.With("method", "NetworkPacketHandler", "socket", socket)

	switch packet.Head {
	case ConnectedNode:
		data, ok := packet.Data.(string)

		if !ok {
			return ErrUnexpectedType
		}

		routing[data] = socket

		logger.Debug("node added in routing table", "uuid", data)

	case RemovedNode:
		data, ok := packet.Data.(string)

		if !ok {
			return ErrUnexpectedType
		}

		delete(routing, data)

		logger.Debug("node removed from routing table", "uuid", data)

	default:
		logger.Warn("unknown network packet recieved")
	}

	return nil
}

func RoutePacket(packet Packet, socket string) error {

	if packet.To == address {
		IncomingPacketHandler(packet)
		return nil
	}

	route, ok := routing[packet.To]
	if ok {
		sockets[route] <- packet
		return nil
	}

	if packet.To == "" {
		err := NetworkPacketHandler(packet, socket)
		if err != nil {
			return err
		}
	}

	if hasDefaultRoute {
		defaultRoute <- packet
		return nil
	}

	return nil
}

func ConnectionHandler(conn net.Conn, socket chan Packet, client string) {

	logger := slog.With("method", "ConnectionHandler")

	wait := new(sync.WaitGroup)
	wait.Add(2)

	go func() {
		decoder := gob.NewDecoder(conn)

		for {
			var packet Packet

			err := decoder.Decode(&packet)
			if err != nil {
				logger.Error("error while decoding packet", "error", err)
				break
			}

			err = RoutePacket(packet, client)
			if err != nil {
				logger.Error("error while routing packet", "error", err)
				break
			}
		}

		close(socket)
		wait.Done()
	}()

	go func() {
		encoder := gob.NewEncoder(conn)

		for {
			message, ok := <-socket
			if !ok {
				break
			}

			err := encoder.Encode(message)
			if err != nil {
				logger.Error("error while encoding message", "error", err)
				break
			}
		}

		err := conn.Close()
		if err != nil {
			logger.Error("error while closing connection", "error", err)
		}

		wait.Done()
	}()

	wait.Wait()
}

func ClientConnection(hostAddress string) {

	handler := func(conn net.Conn) {
		client := uuid.New().String()
		sockets[client] = make(chan Packet, 10)

		ConnectionHandler(conn, sockets[client], client)

		for node, socket := range routing {
			if socket != client {
				continue
			}

			delete(routing, node)

			if !hasDefaultRoute {
				continue
			}

			defaultRoute <- Packet{
				To:   "",
				From: address,
				Head: RemovedNode,
				Data: node,
			}
		}

		delete(sockets, client)

		slog.Info("cleaned up socket")
	}

	logger := slog.With("method", "ClientConnection")

	listen, err := net.Listen("tcp", hostAddress)
	if err != nil {
		logger.Error("error while creating tcp listener", "error", err)
		return
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			logger.Error("error while accepting connection", "error", err)
			return
		}

		go handler(conn)
	}
}

func DefaultRouteConnection(defaultRouteAddress string) {

	logger := slog.With("method", "DefaultRouteConnection")

	conn, err := net.Dial("tcp", defaultRouteAddress)
	if err != nil {
		logger.Error("error while connecting to default route", "error", err)
		return
	}

	defaultRoute = make(chan Packet, 10)

	defaultRoute <- Packet{
		To:   "",
		From: address,
		Head: ConnectedNode,
		Data: address,
	}

	hasDefaultRoute = true

	defer func() {
		hasDefaultRoute = false
		slog.Info("closed default route")
	}()

	ConnectionHandler(conn, defaultRoute, address)
}

func main() {

	var writer io.Writer

	createDefaultRoute := flag.Bool("has-default-route", false, "Set to false to not connect to default route")
	defaultRouteAddress := flag.String("default-route", "127.0.0.1:8000", "Specify default route address")
	hostAddress := flag.String("host-address", "0.0.0.0:8000", "Specify address to listen for connections")
	console := flag.Bool("console", false, "Enable input from console")

	if *console {
		writer = *new(io.Writer)
	} else {
		writer = os.Stdout
	}

	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	routing = make(map[string]string)
	sockets = make(map[string]chan Packet)

	flag.Parse()

	go ClientConnection(*hostAddress)

	if *createDefaultRoute {
		go DefaultRouteConnection(*defaultRouteAddress)
	}

	slog.Info("address generated", "uuid", address)

	if !*console {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
		<-exit

		if hasDefaultRoute {
			close(defaultRoute)
		}

		for _, socket := range sockets {
			close(socket)
		}

		os.Exit(0)
	}

	for {
		fmt.Println("\tSend message")
		fmt.Print("Address: ")

		var to string
		fmt.Scan(&to)

		fmt.Print(">")

		var data string
		fmt.Scanln(&data)

		packet := Packet{
			To:   to,
			From: address,
			Head: GeneralTextMessage,
			Data: data,
		}

		RoutePacket(packet, address)
	}
}
