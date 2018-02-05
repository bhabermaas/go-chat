GO-CHAT

This is a simple chat server with its client. It was developed to understand goroutines
and channels. Unlike most channel examples, this implementation passes structs through
channels instead of just strings. Communication between and client server is through 
JSON packets. Marshalling and unmarshalling JSON between endpoints is demonstrated
in these programs. 

programs:

	imclient.go
	runserver.go
	imserver/imserver.go

To run the server:

	go run runserver.go

To run the client:

	go run imclient.go <user-name>
