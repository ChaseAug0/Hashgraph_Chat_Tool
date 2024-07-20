### README 文档

````markdown
# Hashgraph Consensus Algorithm Implementation

This project is an implementation of the Hashgraph consensus algorithm using Golang. The project includes both the server-side and client-side implementations to manage and maintain a Hashgraph network.

## Features

- **Signal Server**: Uses the Pion WebRTC library to facilitate NAT traversal for clients.
- **WebSocket Service**: Utilizes the Gorilla WebSocket library to establish WebSocket connections.
- **Hashgraph Management**: Maintains the global Hashgraph structure, verifies incoming transactions, and periodically calculates consensus.

## Requirements

- Go 1.15 or later
- MongoDB

## Installation

1. **Clone the repository**:
   ```sh
   git clone https://github.com/YOUR_USERNAME/YOUR_REPOSITORY.git
   cd YOUR_REPOSITORY
   ```
````

2. **Install dependencies**:
   Ensure you have the required Go modules:

   ```sh
   go mod tidy
   ```

3. **Set up MongoDB**:
   Ensure MongoDB is installed and running. Update the MongoDB connection string in the server code if necessary.

## Usage

### Server Side

1. **Start the server**:

   ```sh
   go run main.go
   ```

2. **Server will start on port 8080** and will clear any previous node information from MongoDB.

### Client Side

1. **Run the client**:

   ```sh
   go run hashgraphclient.go
   ```

2. **Send a message**:

   - Enter the message you want to send.
   - Choose the target node from the list of online nodes.

3. **Client will maintain its local Hashgraph** and update it based on received events.

## Project Structure

- `main.go` (server-side): Handles WebSocket connections, node registration, and event forwarding.
- `hashgraph.go` (server-side): Manages the Hashgraph structure, event validation, and consensus calculation.
- `hashgraphclient.go` (client-side): Manages local Hashgraph, connects to the signal server, and handles user input.

## Implementation Details

### Hashgraph Overview

Hashgraph is a distributed ledger technology that achieves consensus without the need for proof-of-work. It uses a directed acyclic graph (DAG) where each vertex (event) represents a transaction or a set of transactions.

### How Hashgraph Works

1. **Events**: Each node creates an event containing transactions, a timestamp, and references to two parent events: `selfParent` and `otherParent`.
2. **Witnesses and Famous Witnesses**:
   - **Witness**: An event is a witness if it is the first event created by a node in a round.
   - **Famous Witness**: A witness that is agreed upon by more than two-thirds of the network.
3. **Rounds**: Events are grouped into rounds. A round increases when more than two-thirds of the network can "see" the previous round's witnesses.
4. **Consensus**: Consensus is reached when an event is seen by more than two-thirds of the network's famous witnesses in a round.

### Business Logic

1. **Server**:

   - Manages WebSocket connections for signaling and data exchange.
   - Registers nodes and assigns unique identifiers.
   - Maintains the global Hashgraph structure.
   - Processes incoming events, validates them, and updates the Hashgraph.
   - Forwards events to target nodes as specified by the client.

2. **Client**:
   - Connects to the server using WebSocket and WebRTC for signaling.
   - Maintains its local Hashgraph and updates it based on events received from the server.
   - Allows users to input transactions and choose target nodes to send events to.
   - Signs events using an ECDSA key pair before sending them.

## How to Use

1. **Start the Server**:

   ```sh
   go run main.go
   ```

   The server will listen on port 8080 and connect to MongoDB to manage node and event data.

2. **Run the Client**:

   ```sh
   go run hashgraphclient.go
   ```

   The client will connect to the server and allow you to input messages and select target nodes.

3. **Send a Message**:

   - Enter the message to send when prompted.
   - Choose the target node from the list of online nodes displayed.

4. **Check Logs**:
   - Server logs will show the registration of nodes, receipt of events, and forwarding of events.
   - Client logs will show the connection status, ICE candidate gathering, and event handling.
