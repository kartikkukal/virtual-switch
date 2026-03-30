# Virtual Switch

A server to route packets between nodes in a network.

# Running
To run the program:

```
go run main.go
```

Or build it:

```
go build main.go
```

# Usage
`--has-default-route`: If set, the server acts as a node and will route packets to the default route. Otherwise, the server acts as a master and route packets between nodes.

`--default-route`: To set the address of the default route.

`--host-address`: To set the address on which to start listening.

`--console`: To enable console input to be able to send messages from the server.

# License
Copyright (C) 2026 Kartik Kukal

 This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see <https://www.gnu.org/licenses/>. 