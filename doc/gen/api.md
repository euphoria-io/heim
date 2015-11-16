# Table of Contents

* [Overview](#overview)
* [Field Types](#field-types)
* [Commands and Replies](#commands-and-replies)
  * [Session Management](#session-management)
    * [auth](#auth)
    * [ping](#ping)
  * [Chat Room Commands](#chat-room-commands)
    * [get-message](#get-message)
    * [log](#log)
    * [nick](#nick)
    * [pm-initiate](#pm-initiate)
    * [send](#send)
    * [who](#who)
  * [Account Management](#account-management)
    * [change-name](#change-name)
    * [change-password](#change-password)
    * [login](#login)
    * [logout](#logout)
    * [register-account](#register-account)
    * [reset-password](#reset-password)
  * [Room Manager Commands](#room-manager-commands)
    * [ban](#ban)
    * [edit-message](#edit-message)
    * [grant-access](#grant-access)
    * [grant-manager](#grant-manager)
    * [revoke-access](#revoke-access)
    * [revoke-manager](#revoke-manager)
    * [unban](#unban)
  * [Staff Commands](#staff-commands)
    * [staff-create-room](#staff-create-room)
    * [staff-grant-manager](#staff-grant-manager)
    * [staff-enroll-otp](#staff-enroll-otp)
    * [staff-invade](#staff-invade)
    * [staff-lock-room](#staff-lock-room)
    * [staff-revoke-access](#staff-revoke-access)
    * [staff-revoke-manager](#staff-revoke-manager)
    * [staff-validate-otp](#staff-validate-otp)
    * [unlock-staff-capability](#unlock-staff-capability)
* [Asynchronous Events](#asynchronous-events)
  * [bounce-event](#bounce-event)
  * [disconnect-event](#disconnect-event)
  * [edit-message-event](#edit-message-event)
  * [hello-event](#hello-event)
  * [join-event](#join-event)
  * [network-event](#network-event)
  * [nick-event](#nick-event)
  * [part-event](#part-event)
  * [ping-event](#ping-event)
  * [pm-initiate-event](#pm-initiate-event)
  * [send-event](#send-event)
  * [snapshot-event](#snapshot-event)

# Overview

Clients interact with Euphoria over a WebSocket-based API. The connection is to a specific
*room*. We call each instance of such a connection a *session*.

## Packets

Messages are sent back and forth between the client and server as packets, in the form of JSON objects.
Each packet has the following structure:

{{template "fields.md" (object "Packet")}}

The `type` field determines the type of the `data` field. Packet types come in three flavors:

1. *Commands*. These names have no suffix. Examples: "[ping](#ping)", "[send](#send)"
2. *Replies*. Every command type has a corresponding reply type. Their names all have a `-reply` suffix. Examples: "[ping-reply](#ping)", "[send-reply](#send)"
3. *Events*. These names all have an `-event` suffix. Examples: "[snapshot-event](#snapshot-event)", "[ping-event](#ping-event)"

Almost all client-to-server packets must be commands. The only exception is [ping-reply](#ping),
which the client should send in response to a [ping-event](#ping-event) from the server.
Any other reply or event sent by the client will have an error reply sent back in response.

All server-to-client packets must be either replies or events. All replies must correspond to a command
sent by the client. The server must never send more than one reply to a command.

When a client sends a command, it can choose to specify an `id`. This is an arbitrary string that
the server will include in its reply. This helps asynchronous clients identify which command a packet
is in reply to.

Here is an example [send](#send) command sent from a client to the server:

```
{
 "id": "1",
 "type": "send",
 "data": {
  "content": "hello world!"
 }
}
```

In response, the server will send back a [send-reply](#send):

```
{
 "id": "1",
 "type": "send-reply",
 "data": {
  "id": "00gd6yy9hvksg",
  "time": 1418585715,
  "sender": {
   "id": "agent:4da8fa7375215589",
   "name": "logan",
   "server_id": "heim.1",
   "server_era": "00g5fdwjzl91c",
   "session_id": "4da8fa7375215589-00000246"
  },
  "content": "hello world!"
 }
}
```

The server will also send a [send-event](#send-event) to all the other sessions connected
to the same room:

```
{
 "type": "send-event",
 "data": {
  "id": "00gd6yy9hvksg",
  "time": 1418585715,
  "sender": {
   "id": "agent:4da8fa7375215589",
   "name": "logan",
   "server_id": "heim.1",
   "server_era": "00g5fdwjzl91c",
   "session_id": "4da8fa7375215589-00000246"
  },
  "content": "hello world!"
 }
}
```

## Initial Handshake

When a client connects to the websocket for a room, the server will begin the session
with a [ping-event](#ping-event):

```
{
 "type": "ping-event",
 "data": {
  "time": 1428979816,
  "next": 1428979846
 }
}
```

The client should immediately reply with the same timestamp:

```
{
 "type": "ping-reply",
 "data": {
  "time": 1428979816
 }
}
```

Once the client replies to the ping, one of two possible events will be sent next.
If the room is a public room, or if the client is logged into an account that has
been granted access to the room, then the server will send a [snapshot-event](#snapshot-event):

```
{
  type: "snapshot-event",
  data: {
    identity: "agent:4da8fa7375215589",
    session_id: "4da8fa7375215589-00000246",
    version: "801ea89a4e410b11410eb61c91971439904e66c0",
    listing: [...],
    log:[...]
  }
}
```

This event serves to fill the client in on recent room history, and lists all the sessions
currently joined in the room. From this point on, the session is *joined* with the room. A
joined session may use chat commands and will receive room events.

If the room is private and the client does not have access, the server will send a
[bounce-event](#bounce-event) instead. At this point the client should obtain the
proper authentication credentials from the user and present them with the [auth](#auth)
or [login](#login) command.

# Field Types

This section describes all the field types one can expect to see in packets.

### bool

A boolean value: `true` or `false`.

### int

A signed 64-bit integer value.

### string

Strings are UTF-8 encoded text. Unless otherwise specified, a string may be of any length.

### object

An arbitrary JSON object.

### AccountView

{{(object "AccountView").Doc}}
{{template "fields.md" (object "AccountView")}}

### AuthOption

`AuthOption` is a string indicating a mode of authentication. It must be one of the
following values:

| Value | Description |
| :-- | :--------- |
| `passcode` | Authentication with a passcode, where a key is derived from the passcode to unlock an access grant. |

### Message

{{(object "Message").Doc}}
{{template "fields.md" (object "Message")}}

### PacketType

`PacketType` is a string describing the type of the packet. For example, "[ping](#ping)",
"[ping-reply](#ping-reply)", and "[ping-event](#ping-event)" are packet types.

### SessionView

{{(object "SessionView").Doc}}
{{template "fields.md" (object "SessionView")}}

### Snowflake

A snowflake is a 13-character string, usually used as a unique identifier for some type
of object. It is the base-36 encoding of an unsigned, 64-bit integer.

### Time

Time is specified as a signed 64-bit integer, giving the number of seconds since the Unix Epoch.

### UserID

A UserID identifies a user. The prefix of this value (up to the colon) indicates a type of session,
while the suffix is a unique value for that type of session.

| Prefix | Suffix | Description |
| :-- | :-- | :----- |
| `agent:` | *agent identifier* | A user, not signed into any account, but tracked via cookie under this identifier. |
| `account:` | *account identifier* | The id ([Snowflake](#snowflake)) of the account the user is logged into. |

# Commands and Replies

As described in the [Overview](#overview), there are a number of commands that the
client may send to the server. For each such command, there is a corresponding reply
that the server will send in return.

## Session Management

Session management commands are involved in the initial handshake and maintenance of a session.

### auth

{{template "command.md" "auth"}}

### ping

{{template "command.md" "ping"}}

## Chat Room Commands

These commands are available to the client once a session successfully joins a room.

### get-message

{{template "command.md" "get-message"}}

### log

{{template "command.md" "log"}}

### nick

{{template "command.md" "nick"}}

### pm-initiate

{{template "command.md" "pm-initiate"}}

### send

{{template "command.md" "send"}}

### who

{{(packet "who").Doc}}

{{(packet "who-reply").Doc}}
{{template "fields.md" (packet "who-reply")}}

## Account Management

These commands enable a client to register, associate, and dissociate with an account.
An account allows an identity to be shared across browsers and devices, and is a
prerequisite for room management.

### change-name

{{template "command.md" "change-name"}}

### change-password

{{template "command.md" "change-password"}}

### login

{{template "command.md" "login"}}

### logout

{{template "command.md" "logout"}}

### register-account

{{template "command.md" "register-account"}}

### reset-password

{{template "command.md" "reset-password"}}

## Room Manager Commands

These commands are available if the client is logged into an account that has a manager grant
on the room.

### ban

{{template "command.md" "ban"}}

### edit-message

{{template "command.md" "edit-message"}}

### grant-access

{{template "command.md" "grant-access"}}

### grant-manager

{{template "command.md" "grant-manager"}}

### revoke-access

{{template "command.md" "revoke-access"}}

### revoke-manager

{{template "command.md" "revoke-manager"}}

### unban

{{template "command.md" "unban"}}

## Staff Commands

Staff commands are only available to site operators. This section is not relevant to
most client implementations.

### staff-create-room

{{template "command.md" "staff-create-room"}}

### staff-enroll-otp

{{template "command.md" "staff-enroll-otp"}}

### staff-grant-manager

{{template "command.md" "staff-grant-manager"}}

### staff-invade

{{template "command.md" "staff-invade"}}

### staff-lock-room

{{template "command.md" "staff-lock-room"}}

### staff-revoke-access

{{template "command.md" "staff-revoke-access"}}

### staff-revoke-manager

{{template "command.md" "staff-revoke-manager"}}

### staff-validate-otp

{{template "command.md" "staff-validate-otp"}}

### unlock-staff-capability

{{template "command.md" "unlock-staff-capability"}}

# Asynchronous Events

The following events may be sent from the server to the client at any time.

## bounce-event

{{(packet "bounce-event").Doc}}
{{template "fields.md" (packet "bounce-event")}}

## disconnect-event

{{(packet "disconnect-event").Doc}}
{{template "fields.md" (packet "disconnect-event")}}

## hello-event

{{(packet "hello-event").Doc}}
{{template "fields.md" (object "HelloEvent")}}

## join-event

A `join-event` indicates a session just joined the room.

{{template "fields.md" (object "PresenceEvent")}}

## network-event

{{(packet "network-event").Doc}}
{{template "fields.md" (packet "network-event")}}

## nick-event

{{(packet "nick-event").Doc}}
{{template "fields.md" (packet "nick-event")}}

## edit-message-event

{{(packet "edit-message-event").Doc}}
{{template "fields.md" (packet "edit-message-event")}}

## part-event

A `part-event` indicates a session just disconnected from the room.

{{template "fields.md" (object "PresenceEvent")}}

## ping-event

{{(packet "ping-event").Doc}}
{{template "fields.md" (packet "ping-event")}}

## pm-initiate-event

{{(packet "pm-initiate-event").Doc}}
{{template "fields.md" (packet "pm-initiate-event")}}

## send-event

{{(packet "send-event").Doc}}
{{template "fields.md" (packet "send-event")}}

## snapshot-event

{{(packet "snapshot-event").Doc}}
{{template "fields.md" (packet "snapshot-event")}}
