var _ = require('lodash')
var url = require('url')
var EventEmitter = require('eventemitter3')


function logPacket(kind, data, highlight) {
  var group = highlight ? 'group' : 'groupCollapsed'
  console[group](
    '%c%s %c%s %c%s',
    kind == 'send' ? 'color: green' : 'color: #06f', kind,
    'color: black', data.type,
    highlight ? 'background: #efb' : 'color: gray; font-weight: normal', data.id ? '(id: ' + data.id + ')' : '(no id)'
  )
  console.log(data)
  console.log(JSON.stringify(data, true, 2))
  console.groupEnd()
}


function Socket() {
  this.endpoint = null
  this.roomName = null
  this.events = new EventEmitter()
  this.ws = null
  this.seq = 0
  this.pingTimeout = null
  this.pingReplyTimeout = null
  this.nextPing = 0
  this.lastMessage = null
  this._logPackets = false
  this._logPacketIds = {}
}

_.extend(Socket.prototype, {
  pingLimit: 2000,

  _wsurl: function(endpoint, roomName) {
    var parsedEndpoint = url.parse(endpoint)

    var prefix = parsedEndpoint.pathname == '/' ? '' : parsedEndpoint.pathname

    var scheme = 'ws'
    if (parsedEndpoint.protocol == 'https:') {
      scheme = 'wss'
    }

    return scheme + '://' + parsedEndpoint.host + prefix + '/room/' + roomName + '/ws?h=1'
  },

  connect: function(endpoint, roomName, opts) {
    this._logPackets = opts && opts.log
    this.endpoint = endpoint
    this.roomName = roomName
    this.reconnect()
  },

  reconnect: function() {
    if (this.ws) {
      // forcefully drop websocket and reconnect
      this._onClose()
      this.ws.close()
    }
    var wsurl = this._wsurl(this.endpoint, this.roomName)
    this.ws = new WebSocket(wsurl, 'heim1')
    this.ws.onopen = this._onOpen.bind(this)
    this.ws.onclose = this._onCloseReconnectSlow.bind(this)
    this.ws.onmessage = this._onMessage.bind(this)
  },

  _onOpen: function() {
    this.events.emit('open')
  },

  _onClose: function() {
    clearTimeout(this.pingTimeout)
    clearTimeout(this.pingReplyTimeout)
    this.pingReplyTimeout = null
    this.ws.onopen = this.ws.onclose = this.ws.onmessage = null
    this.events.emit('close')
  },

  _onCloseReconnectSlow: function() {
    this._onClose()
    var delay = 2000 + 3000 * Math.random()
    setTimeout(this.reconnect.bind(this), delay)
  },

  _onMessage: function(ev) {
    var data = JSON.parse(ev.data)

    var packetLogged = _.has(this._logPacketIds, data.id)
    if (this._logPackets || packetLogged) {
      logPacket('recv', data, packetLogged)
    }

    this.lastMessage = Date.now()

    this._handlePings(data)

    this.events.emit('receive', data)
  },

  _handlePings: function(msg) {
    if (msg.type == 'ping-event') {
      if (msg.data.next > this.nextPing) {
        var interval = msg.data.next - msg.data.time
        this.nextPing = msg.data.next
        clearTimeout(this.pingTimeout)
        this.pingTimeout = setTimeout(this._ping.bind(this), interval * 1000)
      }

      this.send({
        type: 'ping-reply',
        data: {
          time: msg.data.time,
        },
      })
    }

    // receiving any message removes the need to ping
    clearTimeout(this.pingReplyTimeout)
    this.pingReplyTimeout = null
  },

  send: function(data, log) {
    if (!data.id) {
      data.id = String(this.seq++)
    }

    // FIXME: remove when fixed on server
    if (!data.data) {
      data.data = {}
    }

    if (log) {
      this._logPacketIds[data.id] = true
    }
    if (this._logPackets || log) {
      logPacket('send', data, log)
    }
    this.ws.send(JSON.stringify(data))
  },

  _ping: function() {
    if (this.pingReplyTimeout) {
      return
    }

    this.send({
      type: 'ping',
    })

    this.pingReplyTimeout = setTimeout(this.reconnect.bind(this), this.pingLimit)
  },

  pingIfIdle: function() {
    if (this.lastMessage === null || Date.now() - this.lastMessage >= this.pingLimit) {
      this._ping()
    }
  },
})

_.each(['on', 'off', 'once'], function(method) {
  Socket.prototype[method] = function() {
    this.events[method].apply(this.events, arguments)
  }
})

module.exports = Socket
