var _ = require('lodash')
var Reflux = require('reflux')


var storeActions = Reflux.createActions([
  'send',
  'connect',
])
_.extend(module.exports, storeActions)

storeActions.connect.sync = true

module.exports.store = Reflux.createStore({
  listenables: storeActions,

  init: function() {
    this.ws = null
    this.seq = 0
  },

  connect: function(roomName) {
    this.roomName = roomName
    this.ws = new WebSocket(this._wsurl(location, this.roomName), 'heim1')
    this.ws.onopen = this._open
    this.ws.onclose = this._close
    this.ws.onmessage = this._message
    this.connected = true
  },

  _wsurl: function(loc, roomName) {
    var scheme = 'ws'
    if (loc.protocol == 'https:') {
      scheme = 'wss'
    }
    return scheme + '://' + loc.host + '/room/' + roomName + '/ws'
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
      var delay = 2000 + 3000 * Math.random()
      setTimeout(_.partial(this.connect, this.roomName), delay)
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
