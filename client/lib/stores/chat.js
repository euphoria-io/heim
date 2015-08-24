var _ = require('lodash')
var React = require('react/addons')
var Reflux = require('reflux')
var Immutable = require('immutable')

var actions = require('../actions')
var ChatTree = require('../chat-tree')
var storage = require('./storage')
var activity = require('./activity')
var socket = require('./socket')
var plugins = require('./plugins')
var hueHash = require('../hue-hash')


var mentionRe = module.exports.mentionRe = /\B@([^\s]+?(?=$|[,.!?;&'\s]|&#39;|&quot;|&amp;))/g

var storeActions = module.exports.actions = Reflux.createActions([
  'messageReceived',
  'logsReceived',
  'messagesChanged',
  'setRoomSettings',
  'markMessagesSeen',
  'setSelected',
  'deselectAll',
])
storeActions.setRoomSettings.sync = true
storeActions.messagesChanged.sync = true
_.extend(module.exports, storeActions)

module.exports.store = Reflux.createStore({
  listenables: [
    actions,
    storeActions,
    {socketEvent: socket.store},
    {storageChange: storage.store},
    {activityChange: activity.store},
    {onActive: activity.becameActive},
  ],

  seenTTL: 12 * 60 * 60 * 1000,

  init: function() {
    this.state = {
      serverVersion: null,
      id: null,
      sessionId: null,
      connected: null,  // => socket connected
      canJoin: null,
      joined: false,  // => received snapshot; sent nick; ui ready
      loadingLogs: false,
      roomName: null,
      roomSettings: Immutable.Map(),
      tentativeNick: null,
      nick: null,
      authType: null,
      authState: null,
      authData: null,
      isManager: null,
      isStaff: null,
      messages: new ChatTree(),
      earliestLog: null,
      nickHues: {},
      who: Immutable.Map(),
      bannedIds: Immutable.Set(),
      selectedMessages: Immutable.Set(),
    }

    this._loadingLogs = false
    this._seenMessages = Immutable.Map()
    this._joinWhenReady = false

    this.lastActive = null
    this.lastVisit = null

    this.state.messages.changes.on('__all', ids => {
      storeActions.messagesChanged(ids, this.state)
    })

    this._resetLoadingLogsDebounced = _.debounce(this._resetLoadingLogs, 250)
  },

  getInitialState: function() {
    return this.state
  },

  socketEvent: function(ev) {
    // jshint camelcase: false
    if (ev.status == 'receive') {
      if (ev.body.type == 'send-event' || ev.body.type == 'send-reply') {
        this._handleMessageUpdate(ev.body.data, true)
      } else if (ev.body.type == 'edit-message-event') {
        this._handleMessageUpdate(ev.body.data, false)
      } else if (ev.body.type == 'edit-message-reply') {
        if (!ev.body.error) {
          this._handleMessageUpdate(ev.body.data, false)
        } else {
          console.warn('error editing message:', ev.body.error)
        }
      } else if (ev.body.type == 'hello-event') {
        this.state.id = ev.body.data.id
        this.state.isManager = ev.body.data.is_manager
        this.state.isStaff = ev.body.data.is_staff
        if (ev.body.data.room_is_private) {
          this.state.authType = 'passcode'
        }
      } else if (ev.body.type == 'snapshot-event') {
        this.state.serverVersion = ev.body.data.version
        this.state.sessionId = ev.body.data.session_id
        this._joinReady()
        this._handleWhoReply(ev.body.data)
        this._handleLogReply(ev.body.data)
      } else if (ev.body.type == 'bounce-event') {
        this.state.canJoin = false

        var reason = ev.body.data.reason
        if (reason == 'authentication required') {
          this.state.authType = 'passcode'
          if (this.state.authState != 'trying-stored') {
            this.state.authState = 'needs-passcode'
          }
        } else if (reason == 'room not open') {
          this.state.authType = this.state.authState = 'closed'
        }
      } else if (ev.body.type == 'auth-reply') {
        this._handleAuthReply(ev.body.error, ev.body.data)
      } else if (ev.body.type == 'log-reply' && ev.body.data) {
        this._handleLogReply(ev.body.data)
      } else if (ev.body.type == 'who-reply') {
        this._handleWhoReply(ev.body.data)
      } else if (ev.body.type == 'nick-reply' || ev.body.type == 'nick-event') {
        if (ev.body.type == 'nick-reply') {
          this._handleNickReply(ev.body.error, ev.body.data)
        }
        if (!ev.body.error) {
          this.state.who = this.state.who
            .mergeIn([ev.body.data.session_id], {
              session_id: ev.body.data.session_id,
              name: ev.body.data.to,
              hue: hueHash.hue(ev.body.data.to),
            })
        }
      } else if (ev.body.type == 'join-event') {
        ev.body.data.hue = hueHash.hue(ev.body.data.name)
        this.state.who = this.state.who
          .set(ev.body.data.session_id, Immutable.fromJS(ev.body.data))
      } else if (ev.body.type == 'part-event') {
        this.state.who = this.state.who.delete(ev.body.data.session_id)
      } else if (ev.body.type == 'network-event' && ev.body.data.type == 'partition') {
        var id = ev.body.data.server_id
        var era = ev.body.data.server_era
        this.state.who = this.state.who.filterNot(v => v.get('server_id') == id && v.get('server_era') == era)
      } else if (ev.body.type == 'ban-reply') {
        if (!ev.body.error) {
          this.state.bannedIds = this.state.bannedIds.add(ev.body.data.id)
        } else {
          console.warn('error banning:', ev.body.error)
        }
      } else if (ev.body.type == 'ping-event' || ev.body.type == 'ping-reply') {
        // ignore
        return
      } else {
        console.warn('received unknown event of type:', ev.body.type)
      }
    } else if (ev.status == 'open') {
      this.state.connected = true
      if (this.state.authType == 'passcode' && this.state.authData) {
        this._sendPasscode(this.state.authData)
        this.state.authState = 'trying-stored'
      }
    } else if (ev.status == 'close') {
      this.state.connected = false
      this.state.joined = false
      this.state.canJoin = false
    } else {
      console.warn('unexpected socket store status:', ev.status)
    }
    this.trigger(this.state)
  },

  _handleMessageUpdate: function(message, received) {
    var processedMessages = this._handleMessagesData([message])
    this.state.messages.add(processedMessages)
    if (received) {
      _.each(processedMessages, message =>
        storeActions.messageReceived(this.state.messages.get(message.id), this.state)
      )
    }
  },

  _handleMessagesData: function(messages) {
    var seenCutoff = Date.now() - this.seenTTL
    var nick = this.state.nick || this.state.tentativeNick

    this.state.who = this.state.who.withMutations(who => {
      _.each(messages, message => {
        // jshint camelcase: false
        if (nick) {
          var mention = message.content.match(mentionRe)
          if (mention && _.any(mention, m => hueHash.normalize(m.substr(1)) == hueHash.normalize(nick))) {
            message._mention = true
          }
        }
        message.sender.hue = hueHash.hue(message.sender.name)
        who.mergeIn([message.sender.session_id], {
          lastSent: message.time
        })

        if (message.sender.id == this.state.id) {
          message._own = true
        }

        if (!message.parent) {
          delete message.parent
        }

        if (message.time * 1000 < seenCutoff) {
          message._seen = true
        } else {
          var seen = this._seenMessages.get(message.id)
          message._seen = seen ? seen : false
        }
      })
    })

    plugins.hooks.run('incoming-messages', null, messages)
    return messages
  },

  _resetLoadingLogs: function() {
    this.state.loadingLogs = false
    this.trigger(this.state)
  },

  _handleLogReply: function(data) {
    this._loadingLogs = false
    this._resetLoadingLogsDebounced()
    if (!data.log.length) {
      if (data.before) {
        this.state.earliestLog = false
      }
      return
    }
    this.state.earliestLog = data.log[0].id
    React.addons.batchedUpdates(() => {
      var log = this._handleMessagesData(data.log)

      if (!data.before) {
        // persist local tree data but reset out server state
        var shadows = []
        this.state.messages.mapDFS(node => {
          var shadow = node.filter((v, k) => /^_/.test(k))
          if (shadow.size) {
            shadow = shadow.toJS()
            shadow.id = node.get('id')
            shadow.parent = null
            shadows.push(shadow)
          }
        })

        var lastVisit = this.state.messages.get('__lastVisit')
        if (lastVisit) {
          shadows.push(lastVisit.toJS())
        }

        this.state.messages.reset(shadows)
      }
      this.state.messages.add(log)
      storeActions.logsReceived(_.map(log, m => m.id), this.state)
      this.trigger(this.state)
    })
  },

  _handleWhoReply: function(data) {
    // TODO: merge instead of reset so we don't lose lastSent
    this.state.who = Immutable.OrderedMap(
      Immutable.Seq(data.listing)
        .map(function(user) {
          // jshint camelcase: false
          user.hue = hueHash.hue(user.name)
          return [user.session_id, Immutable.Map(user)]
        }, this)
    )
  },

  _handleNickReply: function(error, data) {
    if (!error) {
      this.state.nick = data.to
    }
    delete this.state.tentativeNick
    storage.setRoom(this.state.roomName, 'nick', this.state.nick)
    Raven.setUserContext({
      'id': this.state.id,
      'nick': this.state.nick,
      'session_id': this.state.sessionId,
    })
  },

  _handleAuthReply: function(error, data) {
    if (!error && data.success) {
      this.state.authState = null
      storage.setRoom(this.state.roomName, 'auth', {
        type: this.state.authType,
        data: this.state.authData,
      })
    } else {
      if (this.state.authState == 'trying-stored') {
        this.state.authState = 'needs-passcode'
      } else {
        this.state.authState = 'failed'
      }
    }
  },

  _joinReady: function() {
    this.state.canJoin = true
    if (this._joinWhenReady) {
      this._joinRoom()
    }
  },

  _joinRoom: function() {
    if (!this.state.joined && this.state.canJoin) {
      if (this.state.tentativeNick || this.state.nick) {
        this._sendNick(this.state.tentativeNick || this.state.nick)
      }

      if (!this.state.authType) {
        this.state.authType = 'public'
      }

      this.state.authState = null
      this.state.joined = Date.now()
    }
  },

  storageChange: function(data) {
    if (!data) {
      return
    }
    var roomStorage = data.room[this.state.roomName] || {}
    if (!this.state.nick) {
      this.state.tentativeNick = roomStorage.nick
    }
    if (roomStorage.auth) {
      this.state.authType = roomStorage.auth.type
      this.state.authData = roomStorage.auth.data
    }
    this._seenMessages = Immutable.Map(roomStorage.seenMessages || {})
    this.trigger(this.state)
  },

  activityChange: function(data) {
    this.lastActive = data.lastActive[this.state.roomName]
    if (data.lastVisit[this.state.roomName] != this.lastVisit) {
      this.lastVisit = data.lastVisit[this.state.roomName]
      this.state.messages.add({
        id: '__lastVisit',
        time: this.lastVisit / 1000,
        content: 'last visit',
      })
    }
  },

  onActive: function() {
    if (this.state.connected) {
      socket.pingIfIdle()
    }
  },

  connect: function(roomName, connect) {
    socket.connect(roomName, connect)
    this.state.roomName = roomName
    storage.load()
    this.trigger(this.state)
  },

  joinRoom: function() {
    this._joinWhenReady = true
    this._joinRoom()
    this.trigger(this.state)
  },

  setNick: function(nick) {
    if (nick == this.state.nick || nick == this.state.tentativeNick) {
      return
    }
    this.state.tentativeNick = nick
    this.trigger(this.state)
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

  _sendPasscode: function(passcode) {
    this._authSendId = socket.send({
      type: 'auth',
      data: {
        type: 'passcode',
        passcode: passcode,
      },
    })
  },

  tryRoomPasscode: function(passcode) {
    this.state.authData = passcode
    this.state.authState = 'trying'
    this._sendPasscode(passcode)
    this.trigger(this.state)
  },

  setRoomSettings: function(settings) {
    this.state.roomSettings = this.state.roomSettings.merge(settings)
    this.trigger(this.state)
  },

  loadMoreLogs: function() {
    if (this.state.authState || !this.state.earliestLog || this._loadingLogs) {
      return
    }

    this._resetLoadingLogsDebounced.cancel()
    this._loadingLogs = true
    this.state.loadingLogs = true
    this.trigger(this.state)

    socket.send({
      type: 'log',
      data: {n: 50, before: this.state.earliestLog},
    })
  },

  markMessagesSeen: function(ids) {
    var now = Date.now()

    var unseen = Immutable.Seq(ids)
      .filterNot(id => this.state.messages.get(id).get('_seen'))
      .cacheResult()

    this.state.messages.mergeNodes(unseen.toJS(), {_seen: now})

    var expireThreshold = now - this.seenTTL
    var seenMessages = unseen
      .map(id => [id, now])
      .fromEntrySeq()
      .concat(this._seenMessages.filterNot(ts => ts < expireThreshold))

    if (!Immutable.is(seenMessages, this._seenMessages)) {
      storage.setRoom(this.state.roomName, 'seenMessages', seenMessages.toJS())
    }
  },

  setSelected: function(id, value) {
    this.state.messages.mergeNodes(id, {_selected: value})
    this.state.selectedMessages = this.state.selectedMessages[value ? 'add' : 'delete'](id)
    this.trigger(this.state)
  },

  deselectAll: function() {
    this.state.messages.mergeNodes(this.state.selectedMessages.toArray(), {_selected: false})
    this.state.selectedMessages = this.state.selectedMessages.clear()
    this.trigger(this.state)
  },

  sendMessage: function(content, parent) {
    socket.send({
      type: 'send',
      data: {
        content: content,
        parent: parent || null,
      },
    })
  },
})
