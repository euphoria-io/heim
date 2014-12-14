var _ = require('lodash')
var Reflux = require('reflux')


var actions = Reflux.createActions([
  'send',
  'connect',
])
_.extend(module.exports, actions)

module.exports.store = Reflux.createStore({
  listenables: actions,

  init: function() {
    this.ws = null
  },

  connect: function() {
    var url = 'ws:' + location.host + location.pathname + 'ws'
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

  send: function(data) {
    this.ws.send(JSON.stringify(data))
  }
})
