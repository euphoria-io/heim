var Reflux = require('reflux')


module.exports = Reflux.createStore({
  listenables: [
    require('../actions'),
  ],

  init: function() {
    this.s = null
  },

  connect: function(roomName) {
    var url = 'ws:' + location.host + '/room/' + roomName + '/ws'
    this.s = new WebSocket(url, 'heim1')
    this.s.onopen = this._open
    this.s.onclose = this._close
    this.s.onmessage = this._message
  },

  _open: function() {
    this.trigger({
      status: 'open',
    })
  },

  _close: function() {
    this.trigger({
      status: 'close',
    })
  },

  _message: function(ev) {
    var data = JSON.parse(ev.data)
    this.trigger({
      status: 'receive',
      body: data,
    })
  },

  send: function(data) {
  }
})
