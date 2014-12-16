var _ = require('lodash')
var Reflux = require('reflux')
var Immutable = require('immutable')

var storage = require('./storage')
var socket = require('./socket')


module.exports.store = Reflux.createStore({
  listenables: [
    require('../actions'),
    {socketEvent: socket.store},
    {storageChange: storage.store},
  ],

  init: function() {
    this.state = {
      connected: null,
      messages: Immutable.List(),
      nickHues: {},
      who: Immutable.List(),
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
        this.state.messages = this.state.messages.push(Immutable.fromJS(ev.body.data))

      } else if (ev.body.type == 'log-reply' && ev.body.data) {
        this.state.messages = Immutable.Seq(ev.body.data.log)
          .map(function(message) {
            message.sender.hue = this._getNickHue(message.sender.name)
            return Immutable.fromJS(message)
          }, this)
          .toList()

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
    return this.state.nickHues[nick] = val % 255
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

  sendMessage: function(content) {
    socket.send({
      type: 'send',
      data: {
        content: content
      },
    })
  },
})
