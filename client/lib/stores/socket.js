var Reflux = require('reflux')


module.exports = Reflux.createStore({
  listenables: [
    require('../actions'),
  ],

  init: function() {
    this.ws = null
  },

  connect: function(roomName) {
    var url = 'ws:' + location.host + '/room/' + roomName + '/ws'
    this.ws = new WebSocket(url, 'heim1')
    this.ws.onopen = this._open
    this.ws.onclose = this._close
    this.ws.onmessage = this._message
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

  _send: function(data) {
    this.ws.send(JSON.stringify(data))
  },

  send: function(content) {
    this._send({
      type: 'send',
      data: {
        content: content
      }
    })
  },
})
