import _ from 'lodash'
import ReactDOM from 'react-dom'
import Reflux from 'reflux'
import Immutable from 'immutable'

import actions from '../actions'
import ChatTree from '../chat-tree'
import storage from './storage'
import activity from './activity'
import plugins from './plugins'
import hueHash from '../hue-hash'


const mentionDelim = String.raw`^|$|[,.!?;&'"\s]|&#39;|&quot;|&amp;`
const mentionFindRe = module.exports.mentionFindRe = new RegExp('(' + mentionDelim + String.raw`)@(\S+?)(?=` + mentionDelim + ')', 'g')

const storeActions = module.exports.actions = Reflux.createActions([
  'messageReceived',
  'logsReceived',
  'messagesChanged',
  'setRoomSettings',
  'markMessagesSeen',
  'setSelected',
  'deselectAll',
  'editMessage',
  'banUser',
])
storeActions.setRoomSettings.sync = true
storeActions.messagesChanged.sync = true
_.extend(module.exports, storeActions)

module.exports.store = Reflux.createStore({
  listenables: [
    actions,
    storeActions,
    {storageChange: storage.store},
    {activityChange: activity.store},
    {onActive: activity.becameActive},
  ],

  seenTTL: 12 * 60 * 60 * 1000,

  init() {
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

    this.socket = null

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

  getInitialState() {
    return this.state
  },

  socketOpen() {
    this.state.connected = true
    if (this.state.authType === 'passcode' && this.state.authData) {
      this._sendPasscode(this.state.authData)
      this.state.authState = 'trying-stored'
    }
    this.trigger(this.state)
  },

  socketClose() {
    this.state.connected = false
    this.state.joined = false
    this.state.canJoin = false
    this.trigger(this.state)
  },

  socketEvent(ev) {
    if (ev.type === 'send-event') {
      this._handleMessageUpdate(ev.data, true)
    } else if (ev.type === 'send-reply') {
      if (!ev.error) {
        this._handleMessageUpdate(ev.data, true)
      } else {
        console.warn('error sending message:', ev.error)  // eslint-disable-line no-console
      }
    } else if (ev.type === 'edit-message-event') {
      this._handleMessageUpdate(ev.data, false)
    } else if (ev.type === 'edit-message-reply') {
      if (!ev.error) {
        this._handleMessageUpdate(ev.data, false)
      } else {
        console.warn('error editing message:', ev.error)  // eslint-disable-line no-console
      }
    } else if (ev.type === 'hello-event') {
      this.state.id = ev.data.session.id
      this.state.isManager = ev.data.session.is_manager
      this.state.isStaff = ev.data.session.is_staff
      this.state.authType = ev.data.room_is_private ? 'passcode' : 'public'
      if (ev.data.account_has_access) {
        // note: if there was a stored passcode, we could have an outgoing
        // auth event and authState === 'trying-stored'
        this.state.authState = null
      }
    } else if (ev.type === 'snapshot-event') {
      this.state.serverVersion = ev.data.version
      this.state.sessionId = ev.data.session_id
      this._joinReady()
      this._handleWhoReply(ev.data)
      this._handleLogReply(ev.data)
    } else if (ev.type === 'bounce-event') {
      this.state.canJoin = false

      const reason = ev.data.reason
      if (reason === 'authentication required') {
        this.state.authType = 'passcode'
        if (this.state.authState !== 'trying-stored') {
          this.state.authState = 'needs-passcode'
        }
      } else if (reason === 'room not open') {
        this.state.authType = this.state.authState = 'closed'
      }
    } else if (ev.type === 'auth-reply') {
      this._handleAuthReply(ev.error, ev.data)
    } else if (ev.type === 'log-reply' && ev.data) {
      this._handleLogReply(ev.data)
    } else if (ev.type === 'who-reply') {
      this._handleWhoReply(ev.data)
    } else if (ev.type === 'nick-reply' || ev.type === 'nick-event') {
      if (ev.type === 'nick-reply') {
        this._handleNickReply(ev.error, ev.data)
      }
      if (!ev.error) {
        this.state.who = this.state.who
          .mergeIn([ev.data.session_id], {
            session_id: ev.data.session_id,
            name: ev.data.to,
            hue: hueHash.hue(ev.data.to),
          })
      }
    } else if (ev.type === 'join-event') {
      ev.data.hue = hueHash.hue(ev.data.name)
      this.state.who = this.state.who
        .set(ev.data.session_id, Immutable.fromJS(ev.data))
    } else if (ev.type === 'part-event') {
      this.state.who = this.state.who.delete(ev.data.session_id)
    } else if (ev.type === 'network-event' && ev.data.type === 'partition') {
      const id = ev.data.server_id
      const era = ev.data.server_era
      this.state.who = this.state.who.filterNot(v => v.get('server_id') === id && v.get('server_era') === era)
    } else if (ev.type === 'ban-reply') {
      if (!ev.error) {
        this.state.bannedIds = this.state.bannedIds.add(ev.data.id)
      } else {
        console.warn('error banning:', ev.error)  // eslint-disable-line no-console
      }
    } else if (ev.type === 'ping-event' || ev.type === 'ping-reply') {
      // ignore
      return
    } else {
      console.warn('received unknown event of type:', ev.type)  // eslint-disable-line no-console
    }
    this.trigger(this.state)
  },

  _handleMessageUpdate(message, received) {
    const processedMessages = this._handleMessagesData([message])
    this.state.messages.add(processedMessages)
    if (received) {
      _.each(processedMessages, m =>
        storeActions.messageReceived(this.state.messages.get(m.id), this.state)
      )
    }
  },

  _handleMessagesData(messages) {
    const seenCutoff = Date.now() - this.seenTTL
    const nick = this.state.nick || this.state.tentativeNick

    this.state.who = this.state.who.withMutations(who => {
      _.each(messages, message => {
        if (nick) {
          const mention = message.content.match(mentionFindRe)
          // Note: we are relying on hueHash.normalize to strip the preceding and following characters from the mention regex match here.
          if (mention && _.any(mention, m => hueHash.normalize(m.substr(1)) === hueHash.normalize(nick))) {
            message._mention = true
          }
        }
        message.sender.hue = hueHash.hue(message.sender.name)
        who.mergeIn([message.sender.session_id], {
          lastSent: message.time,
        })

        if (message.sender.id === this.state.id) {
          message._own = true
        }

        if (!message.parent) {
          delete message.parent
        }

        if (message.time * 1000 < seenCutoff) {
          message._seen = true
        } else {
          const seen = this._seenMessages.get(message.id)
          message._seen = seen ? seen : false
        }
      })
    })

    plugins.hooks.run('incoming-messages', null, messages)
    return messages
  },

  _resetLoadingLogs() {
    this.state.loadingLogs = false
    this.trigger(this.state)
  },

  _handleLogReply(data) {
    this._loadingLogs = false
    this._resetLoadingLogsDebounced()
    if (!data.log.length) {
      if (data.before) {
        this.state.earliestLog = false
      }
      return
    }
    this.state.earliestLog = data.log[0].id
    ReactDOM.unstable_batchedUpdates(() => {
      const log = this._handleMessagesData(data.log)

      if (!data.before) {
        // persist local tree data but reset out server state
        const shadows = []
        this.state.messages.mapDFS(node => {
          let shadow = node.filter((v, k) => /^_/.test(k))
          if (shadow.size) {
            shadow = shadow.toJS()
            shadow.id = node.get('id')
            shadow.parent = null
            shadows.push(shadow)
          }
        })

        const lastVisit = this.state.messages.get('__lastVisit')
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

  _handleWhoReply(data) {
    // TODO: merge instead of reset so we don't lose lastSent
    this.state.who = Immutable.OrderedMap(
      Immutable.Seq(data.listing)
        .map(user => {
          user.hue = hueHash.hue(user.name)
          return [user.session_id, Immutable.Map(user)]
        })
    )
  },

  _handleNickReply(error, data) {
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

  _handleAuthReply(error, data) {
    if (!error && data.success) {
      this.state.authState = null
      storage.setRoom(this.state.roomName, 'auth', {
        type: this.state.authType,
        data: this.state.authData,
      })
    } else {
      if (error === 'already joined') {
        return
      } else if (this.state.authState === 'trying-stored') {
        this.state.authState = 'needs-passcode'
      } else {
        this.state.authState = 'failed'
      }
    }
  },

  _joinReady() {
    this.state.canJoin = true
    if (this._joinWhenReady) {
      this._joinRoom()
    }
  },

  _joinRoom() {
    if (!this.state.joined && this.state.canJoin) {
      if (this.state.tentativeNick || this.state.nick) {
        this._sendNick(this.state.tentativeNick || this.state.nick)
      }

      this.state.authState = null
      this.state.joined = Date.now()
    }
  },

  storageChange(data) {
    if (!data) {
      return
    }
    const roomStorage = data.room[this.state.roomName] || {}
    if (!this.state.nick) {
      this.state.tentativeNick = roomStorage.nick
    }
    if (roomStorage.auth) {
      this.state.authType = roomStorage.auth.type
      this.state.authData = roomStorage.auth.data
    }
    this.setRoomSettings({showAllReplies: roomStorage.showAllReplies})
    this._seenMessages = Immutable.Map(roomStorage.seenMessages || {})
    this.trigger(this.state)
  },

  activityChange(data) {
    this.lastActive = data.lastActive[this.state.roomName]
    if (data.lastVisit[this.state.roomName] !== this.lastVisit) {
      this.lastVisit = data.lastVisit[this.state.roomName]
      this.state.messages.add({
        id: '__lastVisit',
        time: this.lastVisit / 1000,
        content: 'last visit',
      })
    }
  },

  onActive() {
    if (this.state.connected) {
      this.socket.pingIfIdle()
    }
  },

  setup(roomName) {
    this.state.roomName = roomName
    storage.load()
    this.trigger(this.state)
  },

  connect() {
    this.socket.on('open', this.socketOpen)
    this.socket.on('close', this.socketClose)
    this.socket.on('receive', this.socketEvent)
    this.socket.endBuffering()
  },

  joinRoom() {
    this._joinWhenReady = true
    this._joinRoom()
    this.trigger(this.state)
  },

  setNick(nick) {
    if (nick === this.state.nick || nick === this.state.tentativeNick) {
      return
    }
    this.state.tentativeNick = nick
    this.trigger(this.state)
    this._sendNick(nick)
  },

  _sendNick(nick) {
    this.socket.send({
      type: 'nick',
      data: {
        name: nick,
      },
    })
  },

  _sendPasscode(passcode) {
    this._authSendId = this.socket.send({
      type: 'auth',
      data: {
        type: 'passcode',
        passcode: passcode,
      },
    })
  },

  tryRoomPasscode(passcode) {
    this.state.authData = passcode
    this.state.authState = 'trying'
    this._sendPasscode(passcode)
    this.trigger(this.state)
  },

  setRoomSettings(settings) {
    this.state.roomSettings = this.state.roomSettings.merge(settings)
    this.trigger(this.state)
  },

  loadMoreLogs() {
    if (this.state.authState || !this.state.earliestLog || this._loadingLogs) {
      return
    }

    this._resetLoadingLogsDebounced.cancel()
    this._loadingLogs = true
    this.state.loadingLogs = true
    this.trigger(this.state)

    this.socket.send({
      type: 'log',
      data: {n: 50, before: this.state.earliestLog},
    })
  },

  markMessagesSeen(ids) {
    const now = Date.now()

    const unseen = Immutable.Seq(ids)
      .filterNot(id => this.state.messages.get(id).get('_seen'))
      .cacheResult()

    this.state.messages.mergeNodes(unseen.toJS(), {_seen: now})

    const expireThreshold = now - this.seenTTL
    const seenMessages = unseen
      .map(id => [id, now])
      .fromEntrySeq()
      .concat(this._seenMessages.filterNot(ts => ts < expireThreshold))

    if (!Immutable.is(seenMessages, this._seenMessages)) {
      storage.setRoom(this.state.roomName, 'seenMessages', seenMessages.toJS())
    }
  },

  setSelected(id, value) {
    this.state.messages.mergeNodes(id, {_selected: value})
    this.state.selectedMessages = this.state.selectedMessages[value ? 'add' : 'delete'](id)
    this.trigger(this.state)
  },

  deselectAll() {
    this.state.messages.mergeNodes(this.state.selectedMessages.toArray(), {_selected: false})
    this.state.selectedMessages = this.state.selectedMessages.clear()
    this.trigger(this.state)
  },

  sendMessage(content, parent) {
    this.socket.send({
      type: 'send',
      data: {
        content: content,
        parent: parent || null,
      },
    })
  },

  editMessage(id, data) {
    this.socket.send({
      type: 'edit-message',
      data: _.merge(data, {id: id}),
    })
  },

  banUser(id, data) {
    this.socket.send({
      type: 'ban',
      data: _.merge(data, {id: id}),
    })
  },
})
