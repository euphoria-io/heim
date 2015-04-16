var fs = require('fs')
var _ = require('lodash')
var Reflux = require('reflux')
var Immutable = require('immutable')

var actions = require('../actions')
var storage = require('./storage')
var chat = require('./chat')


var favicons = module.exports.favicons = {
  'active': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/favicon-active.png', 'base64'),
  'highlight': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/favicon-highlight.png', 'base64'),
  'disconnected': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/favicon-disconnected.png', 'base64'),
}

var icons = module.exports.icons = {
  'normal': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/icon.png', 'base64'),
  'highlight': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/icon-highlight.png', 'base64'),
}

var storeActions = Reflux.createActions([
  'enablePopups',
  'disablePopups',
  'pausePopupsUntil',
  'setRoomNotificationMode',
])
_.extend(module.exports, storeActions)

storeActions.enablePopups.sync = true

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {storageChange: storage.store},
    {chatStateChange: chat.store},
    {messageReceived: chat.messageReceived},
    {messagesChanged: chat.messagesChanged},
    {focusChange: require('./focus').store},
  ],

  timeout: 3 * 1000,
  mentionTTL: 12 * 60 * 60 * 1000,

  init: function() {
    this.state = {
      popupsEnabled: null,
      popupsSupported: 'Notification' in window,
      popupsPausedUntil: null,
    }

    this.focus = true
    this.connected = false
    this.notifications = {}
    this._roomStorage = null
    this._lastMsgId = null

    if (this.state.popupsSupported) {
      this.state.popupsPermission = Notification.permission == 'granted'
    }

    window.addEventListener('unload', this.clearAllNotifications)
  },

  focusChange: function(state) {
    this.focus = state.windowFocused
    if (this.focus) {
      this.clearAllNotifications()
      this.updateFavicon()
    }
  },

  getInitialState: function() {
    return this.state
  },

  enablePopups: function() {
    if (this.state.popupsPermission) {
      storage.set('notify', true)
      storage.set('notifyPausedUntil', null)
    } else {
      Notification.requestPermission(this.onPermission)
    }
  },

  disablePopups: function() {
    storage.set('notify', false)
    storage.set('notifyPausedUntil', null)
  },

  pausePopupsUntil: function(time) {
    storage.set('notifyPausedUntil', Math.floor(time))
  },

  setRoomNotificationMode: function(roomName, mode) {
    storage.setRoom(roomName, 'notifyMode', mode)
  },

  onPermission: function(permission) {
    this.state.popupsPermission = permission == 'granted'
    if (this.state.popupsPermission) {
      storage.set('notify', true)
    }
    this.trigger(this.state)
  },

  storageChange: function(data) {
    if (!data) {
      return
    }
    this.state.popupsEnabled = data.notify
    this.state.popupsPausedUntil = data.notifyPausedUntil
    this._roomStorage = data.room
    this.trigger(this.state)
  },

  chatStateChange: function(chatState) {
    this.connected = chatState.connected
    this.updateFavicon()
  },

  messageReceived: function(message, state) {
    if (!state.joined) {
      return
    }

    var lastMsg = state.messages.last()
    var lastMsgId
    if (lastMsg) {
      lastMsgId = lastMsg.get('id')
    }

    if (lastMsgId && lastMsgId != this._lastMsgId && !lastMsg.get('mention')) {
      this.notify('new-message', state.roomName, lastMsgId, {
        favicon: favicons.active,
        icon: icons.normal,
        body: lastMsg.getIn(['sender', 'name']) + ': ' + lastMsg.get('content'),
        timeout: this.timeout,
      })
      this._lastMsgId = lastMsgId
    }
  },

  messagesChanged: function(ids, state) {
    if (!state.joined) {
      return
    }

    var now = Date.now()

    var roomData = this._roomStorage[state.roomName]
    var seenMentions = Immutable.Map((roomData && roomData.seenMentions) || {})
      .filterNot(expires => now - expires > 0)

    var mentions = Immutable.Seq(ids)
      .filterNot(id => seenMentions.has(id))
      .map(id => state.messages.get(id))
      .filter(msg => msg.get('mention') && now - (msg.get('time') * 1000) < this.mentionTTL)
      .cacheResult()

    if (mentions.size) {
      var msg = mentions.first()
      this.notify('new-mention', state.roomName, msg.get('id'), {
        favicon: favicons.highlight,
        icon: icons.highlight,
        body: msg.getIn(['sender', 'name']) + ': ' + msg.get('content'),
      })
    }

    var expires = now + this.mentionTTL
    var newSeenMentions = mentions
      .map(msg => [msg.get('id'), expires])
      .fromEntrySeq()
      .concat(seenMentions)

    if (newSeenMentions != seenMentions) {
      storage.setRoom(state.roomName, 'seenMentions', newSeenMentions.toJS())
    }
  },

  updateFavicon: function() {
    if (!window.Heim) {
      // still initializing the global
      return
    } else if (!this.connected) {
      Heim.setFavicon(favicons.disconnected)
    } else {
      var notification = this.notifications['new-mention'] || this.notifications['new-message']
      Heim.setFavicon(notification ? notification.favicon : '/static/favicon.png')
    }
  },

  closePopup: function(name) {
    var notification = this.notifications[name]
    if (!notification) {
      return
    }
    clearTimeout(this.notifications[name].timeout)
    // when we close a notification, its onclose callback will get called
    // async. displaying a new notification can race with this, causing the
    // new notification to be invalidly forgotten.
    if (notification.popup) {
      notification.popup.onclose = null
      notification.popup.close()
    }
    notification.popup = null
  },

  clearNotification: function(name) {
    var notification = this.notifications[name]
    if (!notification) {
      return
    }
    this.closePopup(name)
    delete this.notifications[name]
  },

  clearAllNotifications: function() {
    _.each(this.notifications, (notification, name) => {
      this.clearNotification(name)
    })
  },

  notify: function(name, roomName, messageId, options) {
    if (this.focus) {
      return
    }

    this.clearNotification(name)

    var notification = this.notifications[name] = {}
    notification.favicon = options.favicon
    delete options.favicon
    this.updateFavicon()

    var notifyMode = (this._roomStorage[roomName] || {}).notifyMode || 'mention'
    if (notifyMode == 'none') {
      return
    } else if (name == 'new-message' && notifyMode == 'mention') {
      return
    }

    var popupsPaused = this.state.popupsPausedUntil && Date.now() < this.state.popupsPausedUntil
    if (this.state.popupsPermission && this.state.popupsEnabled && !popupsPaused) {
      var timeoutDuration = options.timeout
      delete options.timeout

      notification.popup = new Notification(roomName, options)
      notification.popup.onclick = function() {
        uiwindow.focus()
        actions.focusMessage(messageId)
      }
      notification.popup.onclose = _.partial(this.closePopup, name)

      if (timeoutDuration) {
        notification.timeout = setTimeout(notification.popup.onclose, timeoutDuration)
      }
    }
  },
})
