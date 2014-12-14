var Reflux = require('reflux')

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
      messages: [],
    }
  },

  getInitialState: function() {
    return this.state
  },

  socketEvent: function(ev) {
    if (ev.status == 'receive') {
      if (ev.body.type == 'send') {
        this.state.messages.push(ev.body.data)
      } else if (ev.body.type == 'log' && ev.body.data) {
        this.state.messages = ev.body.data
      }
    } else if (ev.status == 'open') {
      this.state.connected = true
      socket.send({
        type: 'log',
        data: {n: 1000},
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
  },

  connect: function() {
    socket.connect()
  },

  setNick: function(nick) {
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
