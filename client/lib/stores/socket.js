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
    this.seq = 0
  },

  connect: function() {
    this.ws = new WebSocket(this._wsurl(location), 'heim1')
    this.ws.onopen = this._open
    this.ws.onclose = this._close
    this.ws.onmessage = this._message
    this.connected = true
  },

  _wsurl: function(loc) {
    var scheme = 'ws'
    if (loc.protocol == 'https:') {
      scheme = 'wss'
    }
    return scheme + ':' + loc.host + loc.pathname + 'ws'
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

    if (this.connected) {
      setTimeout(this.connect, 2000 + 3000 * Math.random())
    }
  },

  _message: function(ev) {
    var data = JSON.parse(ev.data)
    this.trigger({
      status: 'receive',
      body: data,
    })
  },

  send: function(data) {
    if (!data.id) {
      data.id = String(this.seq++)
    }

    // FIXME: remove when fixed on server
    if (!data.data) {
      data.data = {}
    }

    this.ws.send(JSON.stringify(data))
  }
})
