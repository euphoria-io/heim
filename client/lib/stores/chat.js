var _ = require('lodash')
var Reflux = require('reflux')
var Immutable = require('immutable')

var actions = require('../actions')
var Tree = require('../tree')
var storage = require('./storage')
var socket = require('./socket')


module.exports.store = Reflux.createStore({
  listenables: [
    actions,
    {socketEvent: socket.store},
    {storageChange: storage.store},
  ],

  init: function() {
    this.state = {
      connected: null,
      nick: null,
      messages: new Tree('time'),
      earliestLog: null,
      nickHues: {},
      who: Immutable.OrderedMap(),
      focusedMessage: null,
      entryText: '',
      entrySelectionStart: null,
      entrySelectionEnd: null,
    }
  },

  getInitialState: function() {
    return this.state
  },

  socketEvent: function(ev) {
    if (ev.status == 'receive') {
      if (ev.body.type == 'send-event' || ev.body.type == 'send-reply') {
        var message = ev.body.data
        message.sender.hue = this._getNickHue(message.sender.name)
        this.state.messages.add(message)
      } else if (ev.body.type == 'snapshot-event') {
        this._handleWhoReply(ev.body.data)
        this._handleLogReply(ev.body.data)
      } else if (ev.body.type == 'log-reply' && ev.body.data) {
        this._handleLogReply(ev.body.data)
      } else if (ev.body.type == 'who-reply') {
        this._handleWhoReply(ev.body.data)
      } else if (ev.body.type == 'nick-reply' || ev.body.type == 'nick-event') {
        if (ev.body.type == 'nick-reply') {
          this._handleNickReply(ev.body.data)
        }
        this.state.who = this.state.who
          .mergeIn([ev.body.data.id], {
            id: ev.body.data.id,
            name: ev.body.data.to,
            hue: this._getNickHue(ev.body.data.to),
          })
          .sortBy(function(user) { return user.get('name').toLowerCase() })
      } else if (ev.body.type == 'join-event') {
        ev.body.data.hue = this._getNickHue(ev.body.data.name)
        this.state.who = this.state.who
          .set(ev.body.data.id, Immutable.fromJS(ev.body.data))
          .sortBy(function(user) { return user.get('name').toLowerCase() })
      } else if (ev.body.type == 'part-event') {
        this.state.who = this.state.who.delete(ev.body.data.id)
      }
    } else if (ev.status == 'open') {
      this.state.connected = true
      if (this.state.nick) {
        this._sendNick(this.state.nick)
      }
    } else if (ev.status == 'close') {
      this.state.connected = false
    }
    this.trigger(this.state)
  },

  _handleLogReply: function(data) {
    if (!data.log.length) {
      return
    }
    this._loadingLogs = false
    this.state.earliestLog = data.log[0].id
    _.each(data.log, function(message) {
      message.sender.hue = this._getNickHue(message.sender.name)
    }, this)
    if (data.before) {
      this.state.messages.addAll(data.log)
    } else {
      this.state.messages.reset(data.log)
    }

    if (this.state.focusedMessage) {
      this.state.messages.mergeNode(this.state.focusedMessage, {entry: true})
    }
  },

  _handleWhoReply: function(data) {
    this.state.who = Immutable.OrderedMap(
      Immutable.Seq(data.listing)
        .sortBy(function(user) { return user.name.toLowerCase() })
        .map(function(user) {
          user.hue = this._getNickHue(user.name)
          return [user.id, Immutable.Map(user)]
        }, this)
    )
  },

  _handleNickReply: function(data) {
    this.state.nick = data.to
    this.state.nickText = data.to
    storage.setRoom(this.state.roomName, 'nick', data.to)
  },

  storageChange: function(data) {
    var roomStorage = data.room[this.state.roomName] || {}
    this.state.nick = roomStorage.nick
    this.trigger(this.state)
  },

  _getNickHue: function(nick) {
    if (_.has(this.state.nickHues, nick)) {
      return this.state.nickHues[nick]
    }

    var val = 0
    for (var i = 0; i < nick.length; i++) {
      val += nick.charCodeAt(i)
    }
    this.state.nickHues[nick] = val % 255

    return this.state.nickHues[nick]
  },

  connect: function(roomName) {
    socket.connect(roomName)
    this.state.roomName = roomName
    storage.load()
    this.trigger(this.state)
  },

  setNick: function(nick) {
    if (nick == this.state.nick) {
      return
    }

    storage.setRoom(this.state.roomName, 'nick', nick)
    this._sendNick(nick)
  },

  _sendNick: function(nick) {
    socket.send({
      type: 'nick',
      data: {
        name: nick
      },
    })
  },

  loadMoreLogs: function() {
    if (!this.state.earliestLog || this._loadingLogs) {
      return
    }

    this._loadingLogs = true

    socket.send({
      type: 'log',
      data: {n: 50, before: this.state.earliestLog},
    })
  },

  focusMessage: function(messageId) {
    if (!this.state.nick) {
      return
    }

    messageId = messageId || null
    if (messageId == this.state.focusedMessage) {
      actions.focusEntry()
      return
    }

    if (this.state.focusedMessage) {
      this.state.messages.mergeNode(this.state.focusedMessage, {entry: false})
    }
    if (messageId) {
      this.state.messages.mergeNode(messageId, {entry: true})
    }
    this.state.focusedMessage = messageId
    this.trigger(this.state)
    actions.focusEntry()
  },

  toggleFocusMessage: function(messageId, parentId) {
    var focusParent
    if (parentId == '__root') {
      parentId = null
      focusParent = this.state.focusedMessage == messageId
    } else {
      focusParent = this.state.focusedMessage != parentId
    }

    if (focusParent) {
      actions.focusMessage(parentId)
    } else {
      actions.focusMessage(messageId)
    }
  },

  setEntryText: function(text, selectionStart, selectionEnd) {
    this.state.entryText = text
    this.state.entrySelectionStart = selectionStart
    this.state.entrySelectionEnd = selectionEnd
    // Note: no need to trigger here as nothing updates from this; this data is
    // used to persist entry state across focus changes.
  },

  sendMessage: function(content, parent) {
    socket.send({
      type: 'send',
      data: {
        content: content,
        parent: parent || null,
      },
    })
  },
})
