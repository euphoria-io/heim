# Table of Contents

* [Overview](#overview)
* [Field Types](#field-types)
* [Commands and Replies](#commands-and-replies)
  * [Session Management](#session-management)
    * [auth](#auth)
    * [ping](#ping)
  * [Chat Room Commands](#chat-room-commands)
    * [log](#log)
    * [nick](#nick)
    * [send](#send)
    * [who](#who)
  * [Account Management](#account-management)
    * [login](#login)
    * [logout](#logout)
    * [register-account](#register-account)
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
    * [staff-lock-room](#staff-lock-room)
    * [staff-revoke-access](#staff-revoke-access)
    * [staff-revoke-manager](#staff-revoke-manager)
    * [unlock-staff-capability](#unlock-staff-capability)
* [Asynchronous Events](#asynchronous-events)
  * [bounce-event](#bounce-event)
  * [disconnect-event](#disconnect-event)
  * [edit-message-event](#edit-message-event)
  * [join-event](#join-event)
  * [network-event](#network-event)
  * [nick-event](#nick-event)
  * [part-event](#part-event)
  * [ping-event](#ping-event)
  * [send-event](#send-event)
  * [snapshot-event](#snapshot-event)

# Overview

Clients interact with Euphoria over a WebSocket-based API. The connection is to a specific
*room*. We call each instance of such a connection a *session*.

## Packets

Messages are sent back and forth between the client and server as packets, in the form of JSON objects.
Each packet has the following structure:


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [string](#string) | *optional* |  client-generated id for associating replies with commands |
| `type` | [PacketType](#packettype) | required |  the name of the command, reply, or event |
| `data` | object | *optional* |  the payload of the command, reply, or event |
| `error` | [string](#string) | *optional* |  this field appears in replies if a command fails |
| `throttled` | [bool](#bool) | *optional* |  this field appears in replies to warn the client that it may be flooding; the client should slow down its command rate |
| `throttled_reason` | [string](#string) | *optional* |  if throttled is true, this field describes why |




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

### AuthOption

`AuthOption` is a string indicating a mode of authentication. It must be one of the
following values:

| Value | Description |
| :-- | :--------- |
| `passcode` | Authentication with a passcode, where a key is derived from the passcode to unlock an access grant. |

### Message

A Message is a node in a Room's Log. It corresponds to a chat message, or
a post, or any broadcasted event in a room that should appear in the log.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [Snowflake](#snowflake) | required |  the id of the message (unique within a room) |
| `parent` | [Snowflake](#snowflake) | *optional* |  the id of the message's parent, or null if top-level |
| `previous_edit_id` | [Snowflake](#snowflake) | *optional* |  the edit id of the most recent edit of this message, or null if it's never been edited |
| `time` | [Time](#time) | required |  the unix timestamp of when the message was posted |
| `sender` | [SessionView](#sessionview) | required |  the view of the sender's session |
| `content` | [string](#string) | required |  the content of the message (client-defined) |
| `encryption_key_id` | [string](#string) | *optional* |  the id of the key that encrypts the message in storage |
| `edited` | [Time](#time) | *optional* |  the unix timestamp of when the message was last edited |
| `deleted` | [Time](#time) | *optional* |  the unix timestamp of when the message was deleted |




### PacketType

`PacketType` is a string describing the type of the packet. For example, "[ping](#ping)",
"[ping-reply](#ping-reply)", and "[ping-event](#ping-event)" are packet types.

### SessionView

SessionView describes a session and its identity.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [UserID](#userid) | required |  the id of an agent or account |
| `name` | [string](#string) | required |  the name-in-use at the time this view was captured |
| `server_id` | [string](#string) | required |  the id of the server that captured this view |
| `server_era` | [string](#string) | required |  the era of the server that captured this view |
| `session_id` | [string](#string) | required |  id of the session, unique across all sessions globally |




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

The `auth` command attempts to join a private room. It should be sent in response
to a `bounce-event` at the beginning of a session.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `type` | [AuthOption](#authoption) | required |  the method of authentication |
| `passcode` | [string](#string) | *optional* |  use this field for `passcode` authentication |





The `auth-reply` packet reports whether the `auth` command succeeded.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `success` | [bool](#bool) | required |  true if authentication succeeded |
| `reason` | [string](#string) | *optional* |  if `success` was false, the reason for failure |







### ping

The `ping` command initiates a client-to-server ping. The server will send
back a `ping-reply` with the same timestamp as soon as possible.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `time` | [Time](#time) | required |  an arbitrary value, intended to be a unix timestamp |





`ping-reply` is a response to a `ping` command or `ping-event`.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `time` | [Time](#time) | *optional* |  the timestamp of the ping being replied to |







## Chat Room Commands

These commands are available to the client once a session successfully joins a room.

### log

The `log` command requests messages from the room's message log. This can be used
to supplement the log provided by `snapshot-event` (for example, when scrolling
back further in history).


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `n` | [int](#int) | required |  maximum number of messages to return (up to 1000) |
| `before` | [Snowflake](#snowflake) | *optional* |  return messages prior to this snowflake |





The `log-reply` packet returns a list of messages from the room's message log.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `log` | [[Message](#message)] | required |  list of messages returned |
| `before` | [Snowflake](#snowflake) | *optional* |  messages prior to this snowflake were returned |







### nick

The `nick` command sets the name you present to the room. This name applies
to all messages sent during this session, until the `nick` command is called
again.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `name` | [string](#string) | required |  the requested name (maximum length 36 bytes) |





`nick-reply` confirms the `nick` command. It returns the session's former
and new names (the server may modify the requested nick).


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `session_id` | [string](#string) | required |  the id of the session this name applies to |
| `id` | [UserID](#userid) | required |  the id of the agent or account logged into the session |
| `from` | [string](#string) | required |  the previous name associated with the session |
| `to` | [string](#string) | required |  the name associated with the session henceforth |







### send

The `send` command sends a message to a room. The session must be
successfully joined with the room. This message will be broadcast to
all sessions joined with the room.

If the room is private, then the message content will be encrypted
before it is stored and broadcast to the rest of the room.

The caller of this command will not receive the corresponding
`send-event`, but will receive the same information in the `send-reply`.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `content` | [string](#string) | required |  the content of the message (client-defined) |
| `parent` | [Snowflake](#snowflake) | *optional* |  the id of the parent message, if any |





`send-reply` returns the message that was sent. This includes the message id,
which was populated by the server.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [Snowflake](#snowflake) | required |  the id of the message (unique within a room) |
| `parent` | [Snowflake](#snowflake) | *optional* |  the id of the message's parent, or null if top-level |
| `previous_edit_id` | [Snowflake](#snowflake) | *optional* |  the edit id of the most recent edit of this message, or null if it's never been edited |
| `time` | [Time](#time) | required |  the unix timestamp of when the message was posted |
| `sender` | [SessionView](#sessionview) | required |  the view of the sender's session |
| `content` | [string](#string) | required |  the content of the message (client-defined) |
| `encryption_key_id` | [string](#string) | *optional* |  the id of the key that encrypts the message in storage |
| `edited` | [Time](#time) | *optional* |  the unix timestamp of when the message was last edited |
| `deleted` | [Time](#time) | *optional* |  the unix timestamp of when the message was deleted |







### who

The `who` command requests a list of sessions currently joined in the room.


The `who-reply` packet lists the sessions currently joined in the room.


This packet has no fields.



## Account Management

These commands enable a client to register, associate, and dissociate with an account.
An account allows an identity to be shared across browsers and devices, and is a
prerequisite for room management.

### login

The `login` command attempts to log an anonymous session into an account.
It will return an error if the session is already logged in.

If the login succeeds, the client should expect to receive a
`disconnect-event` shortly after. The next connection the client makes
will be a logged in session.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `namespace` | [string](#string) | required |  the namespace of a personal identifier |
| `id` | [string](#string) | required |  the id of a personal identifier |
| `password` | [string](#string) | required |  the password for unlocking the account |





The `login-reply` packet returns whether the session successfully logged
into an account.

If this reply returns success, the client should expect to receive a
`disconnect-event` shortly after. The next connection the client makes
will be a logged in session.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `success` | [bool](#bool) | required |  true if the session is now logged in |
| `reason` | [string](#string) | *optional* |  if `success` was false, the reason why |
| `account_id` | [Snowflake](#snowflake) | *optional* |  if `success` was true, the id of the account the session logged into. |







### logout

The `logout` command logs a session out of an account. It will return an error
if the session is not logged in.

If the logout is successful, the client should expect to receive a
`disconnect-event` shortly after. The next connection the client
makes will be a logged out session.


This packet has no fields.




The `logout-reply` packet confirms a logout.


This packet has no fields.






### register-account

The `register-account` command creates a new account and logs into it.
It will return an error if the session is already logged in.

If the account registration succeeds, the client should expect to receive a
`disconnect-event` shortly after. The next connection the client makes will be
a logged in session using the new account.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `namespace` | [string](#string) | required |  the namespace of a personal identifier |
| `id` | [string](#string) | required |  the id of a personal identifier |
| `password` | [string](#string) | required |  the password for unlocking the account |





The `register-account-reply` packet returns whether the new account was
registered.

If this reply returns success, the client should expect to receive a
disconnect-event shortly after. The next connection the client makes
will be a logged in session, using the newly created account.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `success` | [bool](#bool) | required |  true if the session is now logged in |
| `reason` | [string](#string) | *optional* |  if `success` was false, the reason why |
| `account_id` | [Snowflake](#snowflake) | *optional* |  if `success` was true, the id of the account the session logged into. |







## Room Manager Commands

These commands are available if the client is logged into an account that has a manager grant
on the room.

### ban

The `ban` command adds an entry to the room's ban list. Any joined sessions
that match this entry will be disconnected. New sessions matching the entry
will be unable to join the room.

The command is a no-op if an identical entry already exists in the ban list.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [UserID](#userid) | *optional* |  if given, select for the given agent or account |
| `ip` | [string](#string) | *optional* |  if given, select for the given IP address |
| `expires` | [Time](#time) | *optional* |  if given, the ban is temporary up until this time |





The `ban-reply` packet indicates that the `ban` command succeeded.


This packet has no fields.






### edit-message

The `edit-message` command can be used by active room managers to modify the
content or display of a message.

A message deleted by this command is still stored in the database. Deleted
messages may be undeleted by this command. (Messages that have expired from
the database due to the room's retention policy are no longer available and
cannot be restored by this or any command).

If the `announce` field is set to true, then an edit-message-event will be
broadcast to the room.

TODO: support content editing
TODO: support reparenting


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [Snowflake](#snowflake) | required |  the id of the message to edit |
| `previous_edit_id` | [Snowflake](#snowflake) | required |  the `previous_edit_id` of the message; if this does not match, the edit will fail (basic conflict resolution) |
| `parent` | [Snowflake](#snowflake) | *optional* |  the new parent of the message (*not yet implemented*) |
| `content` | [string](#string) | *optional* |  the new content of the message (*not yet implemented*) |
| `delete` | [bool](#bool) | required |  the new deletion status of the message |
| `announce` | [bool](#bool) | required |  if true, broadcast an `edit-message-event` to the room |





`edit-message-reply` returns the id of a successful edit.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `edit_id` | [Snowflake](#snowflake) | required |  the unique id of the edit that was applied |
| `deleted` | [bool](#bool) | *optional* |  the new deletion status of the edited message |







### grant-access

The `grant-access` command may be used by an active manager in a private room
to create a new capability for access. Access may be granted to either a
passcode or an account.

If the room is not private, or if the requested access grant already exists,
an error will be returned.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `account_id` | [Snowflake](#snowflake) | *optional* |  the id of an account to grant access to |
| `passcode` | [string](#string) | *optional* |  a passcode to grant access to; anyone presenting the same passcode can access the room |





`grant-access-reply` confirms that access was granted.


This packet has no fields.






### grant-manager

The `grant-manager` command may be used by an active room manager to make
another account a manager in the same room.

An error is returned if the account can't be found.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `account_id` | [Snowflake](#snowflake) | required |  the id of an account to grant manager status to |





`grant-manager-reply` confirms that manager status was granted.


This packet has no fields.






### revoke-access

The `revoke-access` command disables an access grant to a private room.
The grant may be to an account or to a passcode.

TODO: all live sessions using the revoked grant should be disconnected
TODO: support revocation by capability_id, in case a manager doesn't know the passcode


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `account_id` | [Snowflake](#snowflake) | *optional* |  the id of the account to revoke access from |
| `passcode` | [string](#string) | required |  the passcode to revoke access from |





`revoke-access-reply` confirms that the access grant was revoked.


This packet has no fields.






### revoke-manager

The `revoke-manager` command removes an account as manager of the room.
This command can be applied to oneself, so be careful not to orphan
your room!


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `account_id` | [Snowflake](#snowflake) | required |  the id of the account to remove as manager |





`revoke-manager-reply` confirms that the manager grant was revoked.


This packet has no fields.






### unban

The `unban` command removes an entry from the room's ban list.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [UserID](#userid) | *optional* |  if given, select for the given agent or account |
| `ip` | [string](#string) | *optional* |  if given, select for the given IP address |





The `unban-reply` packet indicates that the `unban` command succeeded.


This packet has no fields.






## Staff Commands

Staff commands are only available to site operators. This section is not relevant to
most client implementations.

### staff-create-room

The `staff-create-room` command creates a new room.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `name` | [string](#string) | required |  the name of the new rom |
| `managers` | [[Snowflake](#snowflake)] | required |  ids of manager accounts for this room (there must be at least one) |
| `private` | [bool](#bool) | *optional* |  if true, create a private room (all managers will be granted access) |





`staff-create-room-reply` returns the outcome of a room creation request.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `success` | [bool](#bool) | required |  whether the room was created |
| `failure_reason` | [string](#string) | *optional* |  if `success` was false, the reason why |







### staff-grant-manager

The `staff-grant-manager` command is a version of the [grant-manager](#grant-manager)
command that is available to staff. The staff account does not need to be a manager
of the room to use this command.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `account_id` | [Snowflake](#snowflake) | required |  the id of an account to grant manager status to |





`staff-grant-manager-reply` confirms that requested manager change was granted.


This packet has no fields.






### staff-lock-room

The `staff-lock-room` command makes a room private. If the room is already private,
then it generates a new message key (which currently invalidates all access grants).


This packet has no fields.




`staff-lock-room-reply` confirms that the room has been made newly private.


This packet has no fields.






### staff-revoke-access

The `staff-revoke-access` command is a version of the [revoke-access](#revoke-access)
command that is available to staff. The staff account does not need to be a manager
of the room to use this command.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `account_id` | [Snowflake](#snowflake) | *optional* |  the id of the account to revoke access from |
| `passcode` | [string](#string) | required |  the passcode to revoke access from |





`staff-revoke-access-reply` confirms that requested access capability was revoked.


This packet has no fields.






### staff-revoke-manager

The `staff-revoke-manager` command is a version of the [revoke-manager](#revoke-access)
command that is available to staff. The staff account does not need to be a manager
of the room to use this command.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `account_id` | [Snowflake](#snowflake) | required |  the id of the account to remove as manager |





`staff-revoke-manager-reply` confirms that requested manager capability was revoked.


This packet has no fields.






### unlock-staff-capability

The `unlock-staff-capability` command may be called by a staff account to gain access to
staff commands.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `password` | [string](#string) | required |  the account's password |





`unlock-staff-capability-reply` returns the outcome of unlocking the staff
capability.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `success` | [bool](#bool) | required |  whether staff capability was unlocked |
| `failure_reason` | [string](#string) | *optional* |  if `success` was false, the reason why |







# Asynchronous Events

The following events may be sent from the server to the client at any time.

## bounce-event

A `bounce-event` indicates that access to a room is denied.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `reason` | [string](#string) | *optional* |  the reason why access was denied |
| `auth_options` | [[AuthOption](#authoption)] | *optional* |  authentication options that may be used; see [auth](#auth) |
| `agent_id` | [string](#string) | *optional* |  internal use only |
| `ip` | [string](#string) | *optional* |  internal use only |




## disconnect-event

A `disconnect-event` indicates that the session is being closed. The client
will subsequently be disconnected.

If the disconnect reason is "authentication changed", the client should
immediately reconnect.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `reason` | [string](#string) | required |  the reason for disconnection |




## join-event

A join-event indicates a session just joined the room.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [UserID](#userid) | required |  the id of an agent or account |
| `name` | [string](#string) | required |  the name-in-use at the time this view was captured |
| `server_id` | [string](#string) | required |  the id of the server that captured this view |
| `server_era` | [string](#string) | required |  the era of the server that captured this view |
| `session_id` | [string](#string) | required |  id of the session, unique across all sessions globally |




## network-event

A `network-event` indicates some server-side event that impacts the presence
of sessions in a room.

If the network event type is `partition`, then this should be treated as
a [part-event](#part-event) for all sessions connected to the same server
id/era combo.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `type` | [string](#string) | required |  the type of network event; for now, always `partition` |
| `server_id` | [string](#string) | required |  the id of the affected server |
| `server_era` | [string](#string) | required |  the era of the affected server |




## nick-event

`nick-event` announces a nick change by another session in the room.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `session_id` | [string](#string) | required |  the id of the session this name applies to |
| `id` | [UserID](#userid) | required |  the id of the agent or account logged into the session |
| `from` | [string](#string) | required |  the previous name associated with the session |
| `to` | [string](#string) | required |  the name associated with the session henceforth |




## edit-message-event

An `edit-message-event` indicates that a message in the room has been
modified or deleted. If the client offers a user interface and the
indicated message is currently displayed, it should update its display
accordingly.

The event packet includes a snapshot of the message post-edit.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [Snowflake](#snowflake) | required |  the id of the message (unique within a room) |
| `parent` | [Snowflake](#snowflake) | *optional* |  the id of the message's parent, or null if top-level |
| `previous_edit_id` | [Snowflake](#snowflake) | *optional* |  the edit id of the most recent edit of this message, or null if it's never been edited |
| `time` | [Time](#time) | required |  the unix timestamp of when the message was posted |
| `sender` | [SessionView](#sessionview) | required |  the view of the sender's session |
| `content` | [string](#string) | required |  the content of the message (client-defined) |
| `encryption_key_id` | [string](#string) | *optional* |  the id of the key that encrypts the message in storage |
| `edited` | [Time](#time) | *optional* |  the unix timestamp of when the message was last edited |
| `deleted` | [Time](#time) | *optional* |  the unix timestamp of when the message was deleted |
| `edit_id` | [Snowflake](#snowflake) | required |  the id of the edit |




## part-event

A part-event indicates a session just disconnected from the room.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [UserID](#userid) | required |  the id of an agent or account |
| `name` | [string](#string) | required |  the name-in-use at the time this view was captured |
| `server_id` | [string](#string) | required |  the id of the server that captured this view |
| `server_era` | [string](#string) | required |  the era of the server that captured this view |
| `session_id` | [string](#string) | required |  id of the session, unique across all sessions globally |




## ping-event

A `ping-event` represents a server-to-client ping. The client should send back
a `ping-reply` with the same value for the time field as soon as possible
(or risk disconnection).


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `time` | [Time](#time) | required |  a unix timestamp according to the server's clock |
| `next` | [Time](#time) | required |  the expected time of the next ping-event, according to the server's clock |




## send-event

A `send-event` indicates a message received by the room from another session.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `id` | [Snowflake](#snowflake) | required |  the id of the message (unique within a room) |
| `parent` | [Snowflake](#snowflake) | *optional* |  the id of the message's parent, or null if top-level |
| `previous_edit_id` | [Snowflake](#snowflake) | *optional* |  the edit id of the most recent edit of this message, or null if it's never been edited |
| `time` | [Time](#time) | required |  the unix timestamp of when the message was posted |
| `sender` | [SessionView](#sessionview) | required |  the view of the sender's session |
| `content` | [string](#string) | required |  the content of the message (client-defined) |
| `encryption_key_id` | [string](#string) | *optional* |  the id of the key that encrypts the message in storage |
| `edited` | [Time](#time) | *optional* |  the unix timestamp of when the message was last edited |
| `deleted` | [Time](#time) | *optional* |  the unix timestamp of when the message was deleted |




## snapshot-event

A `snapshot-event` indicates that a session has successfully joined a room.
It also offers a snapshot of the room's state and recent history.


| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
| `identity` | [UserID](#userid) | required |  the id of the agent or account logged into this session |
| `session_id` | [string](#string) | required |  the globally unique id of this session |
| `version` | [string](#string) | required |  the server's version identifier |
| `listing` | [[SessionView](#sessionview)] | required |  the list of all other sessions joined to the room (excluding this session) |
| `log` | [[Message](#message)] | required |  the most recent messages posted to the room (currently up to 100) |



