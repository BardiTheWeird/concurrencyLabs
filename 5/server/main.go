package main

import "parallel-computations-5/server"

// Сервер розсилає повідомлення в певний час певним клієнтам.
func main() {
	srv := server.InitializeServer()
	srv.Start()
}
