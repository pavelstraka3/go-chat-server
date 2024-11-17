# Go WebSocket Chat Application

This is a real-time chat application built using Go and WebSockets. The application allows multiple clients to connect, register usernames, and join chat rooms. It supports broadcasting messages to all clients, direct messaging between users, and basic chat room functionality.

## Features

- **WebSocket communication**: Clients can communicate with the server using WebSockets for real-time messaging.
- **Username registration**: Each client must provide a unique username to participate in the chat.
- **Chat rooms**: Users can create and join chat rooms to engage in more focused conversations.
- **Direct messaging**: Users can send direct messages to one another.
- **Message history**: Chat messages are stored and broadcast to new users as they connect.

## Prerequisites

- Go 1.18 or higher
- Gorilla WebSocket library
- A terminal or command-line interface for running the server and interacting with the application

## Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/yourusername/go-websocket-chat.git
   cd go-websocket-chat
   ```

2. **Install dependencies**:
   The project uses the `gorilla/websocket` package for WebSocket communication. Install it using:
   ```bash
   go get github.com/gorilla/websocket
   ```

## Running the Application

1. **Start the server**:
   To run the server, execute the following command:
   ```bash
   go run main.go
   ```
   The server will start listening on `ws://localhost:8080`.

2. **Connecting a client**:
   Use a WebSocket client or browser-based WebSocket client (such as `wscat` or any browser dev tools) to connect to the server:
   ```bash
   wscat -c ws://localhost:8080
   ```

3. **Interact with the application**:
   - **Enter a username** when prompted.
   - **Join a chat room** using the `/join <roomName>` command.
   - **Send a message** to the room after joining.
   - **Send direct messages** using the format `/dm <username> <message>`.
   - **View message history** that will be sent to new clients as they join.

## API Endpoints

- `GET /ping`: Displays a "Hello!" message for a quick check.
- **WebSocket**: Connect to the WebSocket server at `ws://localhost:8080`.
   - Upon connection, users are prompted to enter a username.
   - After entering a valid username, users are asked to join a chat room (if applicable).

## How It Works

### 1. **WebSocket Connection**:
   When a client connects, the server first requests the username. After the username is validated and reserved, the client is asked to join a chat room. The client can send messages after successfully joining the room.

### 2. **Message Parsing**:
   The server supports regular messages, direct messages (using the `/dm <username>` format), and basic commands like `/join <roomName>` and `/help` for viewing available usernames.

### 3. **Room Management**:
   - The server creates rooms dynamically as users join with the `/join <roomName>` command.
   - Each room maintains a list of clients who are currently connected.
   - When a user joins a room, other users in that room are notified.

### 4. **Client Management**:
   The `ClientManager` struct manages connected clients, ensuring that usernames are unique and tracking which room each client is in.

## Example Commands

- **Join a room**: `/join room1`
- **Send a message**: `Hello, everyone!`
- **Send a direct message**: `/dm username Hello, how are you?`
- **List active users**: `/help`

## Contributing

Feel free to fork this project and create pull requests. If you have any issues or suggestions, please open an issue in the repository.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
