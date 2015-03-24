var _ = require('lodash')
var Reflux = require('reflux')
var Immutable = require('immutable')

var actions = require('../actions')
var storage = require('./storage')
var chat = require('./chat')


var storeActions = Reflux.createActions([
  'enablePopups',
  'disablePopups',
])
_.extend(module.exports, storeActions)

storeActions.enablePopups.sync = true

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {storageChange: storage.store},
    {messageReceived: chat.messageReceived},
    {messagesChanged: chat.messagesChanged},
    {focusChange: require('./focus').store},
  ],

  timeout: 3 * 1000,
  mentionTTL: 12 * 60 * 60 * 1000,

  init: function() {
    this.state = {
      popupsEnabled: null,
      popupsSupported: 'Notification' in window
    }

    this.focus = true
    this.notifications = {}
    this._roomStorage = null
    this._lastMsgId = null

    if (this.state.popupsSupported) {
      this.state.popupsPermission = Notification.permission == 'granted'
    }

    window.addEventListener('unload', this.closeAllNotifications)
  },

  focusChange: function(state) {
    this.focus = state.windowFocused
    if (this.focus) {
      this.closeAllNotifications()
      this.updateFavicon()
    }
  },

  getInitialState: function() {
    return this.state
  },

  enablePopups: function() {
    if (this.state.popupsPermission) {
      storage.set('notify', true)
    } else {
      Notification.requestPermission(this.onPermission)
    }
  },

  disablePopups: function() {
    storage.set('notify', false)
  },

  onPermission: function(permission) {
    this.state.popupsPermission = permission == 'granted'
    if (this.state.popupsPermission) {
      storage.set('notify', true)
    }
    this.trigger(this.state)
  },

  storageChange: function(data) {
    this.state.popupsEnabled = this.state.popupsPermission && data.notify
    this._roomStorage = data.room
    this.trigger(this.state)
  },

  messageReceived: function(message, state) {
    var lastMsg = state.messages.last()
    var lastMsgId
    if (lastMsg) {
      lastMsgId = lastMsg.get('id')
    }

    if (lastMsgId && lastMsgId != this._lastMsgId && !lastMsg.get('mention')) {
      this.notify('new-message', state.roomName, lastMsgId, {
        favicon: '/static/favicon-active.png',
        icon: '/static/icon.png',
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
        favicon: '/static/favicon-highlight.png',
        icon: '/static/icon-highlight.png',
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
    var notification = this.notifications['new-mention'] || this.notifications['new-message']
    Heim.setFavicon(notification ? notification.favicon : '/static/favicon.png')
  },

  closeNotification: function(name) {
    var notification = this.notifications[name]
    if (notification) {
      // when we close a notification, its onclose callback will get called
      // async. displaying a new notification can race with this, causing the
      // new notification to be invalidly forgotten.
      if (this.state.popupsEnabled) {
        notification.popup.onclose = null
        notification.popup.close()
      }
      this.resetNotification(name)
    }
  },

  closeAllNotifications: function() {
    _.each(this.notifications, (notification, name) => {
      this.closeNotification(name)
    })
  },

  resetNotification: function(name) {
    clearTimeout(this.notifications[name].timeout)
    delete this.notifications[name]
  },

  notify: function(name, message, messageId, options) {
    if (this.focus) {
      return
    }

    this.closeNotification(name)

    var notification = this.notifications[name] = this.notifications[name] || {}
    notification.favicon = options.favicon
    delete options.favicon
    this.updateFavicon()

    if (this.state.popupsEnabled) {
      var timeoutDuration = options.timeout
      delete options.timeout

      notification.popup = new Notification(message, options)
      notification.popup.onclick = function() {
        uiwindow.focus()
        actions.focusMessage(messageId)
      }
      notification.popup.onclose = _.partial(this.resetNotification, name)

      if (timeoutDuration) {
        notification.timeout = setTimeout(_.partial(this.closeNotification, name), timeoutDuration)
      }
    }
  },
})
