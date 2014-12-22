Packets
=======

The heim API is transmitted via websocket in units called *packets*. Each packet is serialized to JSON with UTF-8 encoding for transmission.

Packets originating from the client will always receive a packet from the server in response. If a client sends packets A and B, then the server will always send the response to A before it sends the response to B.

The server may also send packets to the client asynchronously. For example, when a user says something in a room, a packet is broadcast to each other session connected to that room. An asynchronous packet could be transmitted before the response to a client packet is transmitted.

Every packet shares a common top-level structure. Here is an example packet sent by the client when a user says something in a room:

```
{
  id: "1",
  type: "send",
  data: {
    content: "hello ezzie!"
  }
}
```

The `id`, `type`, and `data` fields are common to all packets. The `id` *should* be given in any packet sent by the client, and its value *should* be unique in the lifetime of the connection. The server *must* include the same `id` when it sends the corresponding response packet, if an `id` was given. Asynchronous packets from the server *must not* include an `id`.

The `type` field specifies the command or event being communicated, and this determines the expected structure of the `data` field.

Client Commands
===============

The following commands are available to the client:

| Type | Purpose | Example |
| ----: | :------- | :------- |
| `nick` | Modify the user's display name for this session. | 
| `who` | Receive a listing of live users in the room. |
| `log` | Receive the most recent chat history. |
| `send` | Send a message to the room's chat. |

<h4>nick</h4>

Use the nick command to change the user's display name, for subsequent messages sent from the user and for subsequent listings of the room.

Request: `{id: "1", type: "nick", data: {name: "Ezzie"}}`

Response: `{id: "1", type: "nick", data: {id: "logan", name: "Ezzie"}}`

The response will also be broadcast to all other users in the room.

Broadcast: `{type: "nick", data: {id: "logan", name: "Ezzie", from: "Logan"}}`

<h4>who</h4>

Use the who command to fetch the current list of live users in the room.

Request: `{id: "1", type: "who"}`

Response: `{id: "1", type: "who", data: [{id: "logan", name: "Logan"}, {id: "chromakode", name: "Max"}]}

<h4>log</h4>

Use the log command to fetch the latest *n* messages from the room's chat.

Request: `{id: "1", type: "log", data: {n: 50}}`

Response:

```
{
  id: "1",
  type: "log",
  data: [
    {time: 1418585692, sender: {id: "chromakode", name: "Max"}, content: "hi!"},
    {time: 1418585697, sender: {id: "logan", name: "Logan"}, content: "j0!"}
  ]
}
```

<h4>send</h4>

Use the send command to send a message to the room's chat.

Request: `{id: "1", type: "send", data: {content: "hello ezzie"}}`

Response:
```
{
  id: "1",
  type: "send",
  data: {
    time: 1418585715,
    sender: {id: "logan", name: "Logan"},
    content: "hello ezzie"
  }
}
```

The response will also be broadcast to all other users in the room.

Broadcast:
```
{
  type: "send",
  data: {
    time: 1418585715,
    sender: {id: "logan", name: "Logan"},
    content: "hello ezzie"
  }
}
```
