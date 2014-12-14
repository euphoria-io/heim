var Reflux = require('reflux')

var SocketStore = require('./socket')


module.exports = Reflux.createStore({
  listenables: [
    require('../actions'),
    {socketEvent: require('./socket')},
  ],

  init: function() {
    this.state = {
      connected: false,
      messages: []
    }
  },

  getDefaultData: function() {
    return this.state
  },

  socketEvent: function(ev) {
    if (ev.status == 'receive') {
      if (ev.body.type == 'message') {
        this.state.messages.push(ev.body.data)
      }
    } else if (ev.status == 'open') {
      this.state.connected = true
    } else if (ev.status == 'close') {
      this.state.connected = false
    }
    this.trigger(this.state)
  },
})
