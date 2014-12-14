var Reflux = require('reflux')

var socket = require('./socket')


module.exports = Reflux.createStore({
  listenables: [
    require('../actions'),
    {socketEvent: socket.store},
  ],

  init: function() {
    this.state = {
      connected: null,
      messages: [],
    }
  },

  getDefaultData: function() {
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
    } else if (ev.status == 'close') {
      this.state.connected = false
    }
    this.trigger(this.state)
  },

  connect: function() {
    socket.connect()
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
