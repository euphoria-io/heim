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
      messages: new Tree(),
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

      } else if (ev.body.type == 'log-reply' && ev.body.data) {
        _.each(ev.body.data.log, function(message) {
          message.sender.hue = this._getNickHue(message.sender.name)
        }, this)
        this.state.messages.reset(ev.body.data.log)

      } else if (ev.body.type == 'who-reply') {
        this.state.who = Immutable.OrderedMap(
          Immutable.Seq(ev.body.data.listing)
            .sortBy(function(user) { return user.name })
            .map(function(user) {
              user.hue = this._getNickHue(user.name)
              return [user.id, Immutable.Map(user)]
            }, this)
        )
      } else if (ev.body.type == 'nick-reply' || ev.body.type == 'nick-event') {
        this.state.who = this.state.who
          .mergeIn([ev.body.data.id], {
            name: ev.body.data.to,
            hue: this._getNickHue(ev.body.data.to),
          })
          .sortBy(function(user) { return user.get('name') })
      } else if (ev.body.type == 'join-event') {
        ev.body.data.hue = this._getNickHue(ev.body.data.name)
        this.state.who = this.state.who
          .set(ev.body.data.id, Immutable.fromJS(ev.body.data))
          .sortBy(function(user) { return user.get('name') })
      } else if (ev.body.type == 'part-event') {
        this.state.who = this.state.who.delete(ev.body.data.id)
      }
    } else if (ev.status == 'open') {
      this.state.connected = true
      socket.send({
        type: 'log',
        data: {n: 1000},
      })
      socket.send({
        type: 'who',
      })
      if (this.state.nick) {
        this._sendNick(this.state.nick)
      }
    } else if (ev.status == 'close') {
      this.state.connected = false
    }
    this.trigger(this.state)
  },

  storageChange: function(data) {
    this.state.nick = data.nick
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

  connect: function() {
    socket.connect()
  },

  setNick: function(nick) {
    if (nick == this.state.nick) {
      return
    }

    storage.set('nick', nick)
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

  focusMessage: function(messageId) {
    messageId = messageId || null
    if (!this.state.nick || messageId == this.state.focusedMessage) {
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
    this.trigger(this.state)
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
