var _ = require('lodash')
var Reflux = require('reflux')
var Immutable = require('immutable')
var Trie = require('triejs')

var actions = require('../actions')
var Tree = require('../tree')
var storage = require('./storage')
var socket = require('./socket')
var plugins = require('./plugins')
var hueHash = require('../huehash')


function NickTrie() {
  var trie = new Trie({enableCache: false})
  // strip spaces from nicks going into the trie
  _.each(['add', 'remove', 'contains'], function(n) {
    trie[n] = _.wrap(trie[n], function(f, word) {
      return f.call(this, hueHash.stripSpaces(word))
    })
  })
  return trie
}

var mentionRe = module.exports.mentionRe = /@([^\s]+?(?=$|[,.!;\s]|&#39;|&quot;|&amp;))/g

module.exports.store = Reflux.createStore({
  listenables: [
    actions,
    {socketEvent: socket.store},
    {storageChange: storage.store},
    {focusChange: require('./focus').store},
  ],

  init: function() {
    this.state = {
      serverVersion: null,
      sessionId: null,
      connected: null,  // => socket connected
      canJoin: null,
      joined: false,  // => received snapshot; sent nick; ui ready
      roomName: null,
      tentativeNick: null,
      nick: null,
      authType: null,
      authState: null,
      authData: null,
      messages: new Tree('time'),
      earliestLog: null,
      nickHues: {},
      who: Immutable.Map(),
      nickTrie: NickTrie(),
      focusedMessage: null,
      entryText: '',
      entrySelectionStart: null,
      entrySelectionEnd: null,
    }

    this._joinWhenReady = false
  },

  getInitialState: function() {
    return this.state
  },

  socketEvent: function(ev) {
    // jshint camelcase: false
    if (ev.status == 'receive') {
      if (ev.body.type == 'send-event' || ev.body.type == 'send-reply') {
        var message = ev.body.data
        var processedMessages = this._handleMessagesData([message])
        this.state.messages.addAll(processedMessages)
      } else if (ev.body.type == 'snapshot-event') {
        this.state.serverVersion = ev.body.data.version
        this.state.sessionId = ev.body.data.session_id
        this._handleWhoReply(ev.body.data)
        this._handleLogReply(ev.body.data)
        this._joinReady()
      } else if (ev.body.type == 'bounce-event') {
        this.state.canJoin = false
        this.state.authType = 'passcode'
        if (this.state.authState != 'trying-stored') {
          this.state.authState = 'needs-passcode'
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
            .mergeIn([ev.body.data.id], {
              id: ev.body.data.id,
              name: ev.body.data.to,
              hue: hueHash.hue(ev.body.data.to),
            })
          this.state.nickTrie.remove(ev.body.data.from)
          this.state.nickTrie.add(ev.body.data.to)
        }
      } else if (ev.body.type == 'join-event') {
        ev.body.data.hue = hueHash.hue(ev.body.data.name)
        this.state.who = this.state.who
          .set(ev.body.data.id, Immutable.fromJS(ev.body.data))
        this.state.nickTrie.add(ev.body.data.name)
      } else if (ev.body.type == 'part-event') {
        this.state.who = this.state.who.delete(ev.body.data.id)
        this.state.nickTrie.remove(ev.body.data.name)
      } else if (ev.body.type == 'network-event') {
        if (ev.body.data.type == 'partition') {
          var id = ev.body.data.server_id
          var era = ev.body.data.server_era
          this.state.who = this.state.who.filter(v => {
            if (v.get('server_id') == id && v.get('server_era') == era) {
              this.state.nickTrie.remove(v.get('name'))
              return false
            } else {
              return true
            }
          })
        }
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
    }
    this.trigger(this.state)
  },

  _handleMessagesData: function(messages) {
    this.state.who = this.state.who.withMutations(who => {
      _.each(messages, message => {
        var nick = this.state.nick || this.state.tentativeNick
        if (nick) {
          var mention = message.content.match(mentionRe)
          if (mention && _.any(mention, m => m.substr(1).toLowerCase() == hueHash.stripSpaces(nick).toLowerCase())) {
            message.mention = true
          }
        }
        message.sender.hue = hueHash.hue(message.sender.name)
        who.mergeIn([message.sender.id], {
          lastSent: message.time
        })
      })
    })
    plugins.hooks.run('incoming-messages', null, messages)
    return messages
  },

  _handleLogReply: function(data) {
    if (!data.log.length) {
      return
    }
    this._loadingLogs = false
    this.state.earliestLog = data.log[0].id
    var log = this._handleMessagesData(data.log)
    if (data.before) {
      this.state.messages.addAll(log)
    } else {
      this.state.messages.reset(log)
    }

    if (this.state.focusedMessage) {
      this.state.messages.mergeNode(this.state.focusedMessage, {entry: true})
    }
  },

  _handleWhoReply: function(data) {
    // TODO: merge instead of reset so we don't lose lastSent
    this.state.nickTrie = NickTrie()
    this.state.who = Immutable.OrderedMap(
      Immutable.Seq(data.listing)
        .map(function(user) {
          this.state.nickTrie.add(user.name)
          user.hue = hueHash.hue(user.name)
          return [user.id, Immutable.Map(user)]
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
      'id': this.state.sessionId.split('-')[0],
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
      } else if (this.state.authState == 'trying') {
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
      this.state.joined = true
    }
  },

  storageChange: function(data) {
    var roomStorage = data.room[this.state.roomName] || {}
    if (!this.state.nick) {
      this.state.tentativeNick = roomStorage.nick
    }
    if (roomStorage.auth) {
      this.state.authType = roomStorage.auth.type
      this.state.authData = roomStorage.auth.data
    }
    this.trigger(this.state)
  },

  focusChange: function(focusState) {
    if (focusState.windowFocused && this.state.connected) {
      socket.pingIfIdle()
    }
  },

  connect: function(roomName) {
    socket.connect(roomName)
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

  loadMoreLogs: function() {
    if (!this.state.earliestLog || this._loadingLogs) {
      return
    }

    this._loadingLogs = true

    socket.send({
      type: 'log',
      data: {n: 50, before: this.state.earliestLog},
    })
  },

  focusMessage: function(messageId) {
    if (!this.state.nick) {
      return
    }

    messageId = messageId || null
    if (messageId == this.state.focusedMessage) {
      actions.focusEntry()
      return
    }

    if (this.state.focusedMessage) {
      this.state.messages.mergeNode(this.state.focusedMessage, {entry: false})
    }
    if (messageId) {
      this.state.messages.mergeNode(messageId, {entry: true})
    }
    this.state.focusedMessage = messageId
    this.trigger(this.state)
    actions.focusEntry()
  },

  toggleFocusMessage: function(messageId, parentId) {
    var focusParent
    if (parentId == '__root') {
      parentId = null
      focusParent = this.state.focusedMessage == messageId
    } else {
      focusParent = this.state.focusedMessage != parentId
    }

    if (focusParent) {
      actions.focusMessage(parentId)
    } else {
      actions.focusMessage(messageId)
    }
  },

  setEntryText: function(text, selectionStart, selectionEnd) {
    this.state.entryText = text
    this.state.entrySelectionStart = selectionStart
    this.state.entrySelectionEnd = selectionEnd
    // Note: no need to trigger here as nothing updates from this; this data is
    // used to persist entry state across focus changes.
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
