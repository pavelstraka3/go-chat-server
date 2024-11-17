package main

type ChatRoom struct {
	Name    string
	Clients map[string]*Client
	History []string
}
