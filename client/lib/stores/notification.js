var _ = require('lodash')
var Reflux = require('reflux')

var actions = require('../actions')
var storage = require('./storage')


var storeActions = Reflux.createActions([
  'enable',
  'disable',
])
_.extend(module.exports, storeActions)

storeActions.enable.sync = true

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {storageChange: storage.store},
    {chatUpdate: require('./chat').store},
    {focusChange: require('./focus').store},
  ],

  timeout: 3 * 1000,

  init: function() {
    this.state = {
      enabled: null,
      supported: 'Notification' in window
    }

    this.focus = true
    this.notification = null
    this._lastMsgId = false

    if (this.state.supported) {
      this.state.permission = Notification.permission == 'granted'
    }
  },

  focusChange: function(state) {
    this.focus = state.windowFocused
    if (this.focus) {
      Heim.setFavicon('/static/favicon.png')
      this.closeNotification()
    }
  },

  getInitialState: function() {
    return this.state
  },

  enable: function() {
    if (this.state.permission) {
      storage.set('notify', true)
    } else {
      Notification.requestPermission(this.onPermission)
    }
  },

  disable: function() {
    storage.set('notify', false)
  },

  onPermission: function(permission) {
    this.state.permission = permission == 'granted'
    if (this.state.permission) {
      storage.set('notify', true)
    }
    this.trigger(this.state)
  },

  storageChange: function(data) {
    this.state.enabled = this.state.permission && data.notify
    this.trigger(this.state)
  },

  chatUpdate: function(state) {
    if (!state.joined) {
      return
    }

    var lastMsg = state.messages.last()
    var lastMsgId
    if (lastMsg) {
      lastMsgId = lastMsg.get('id')
    }

    if (this._lastMsgId === false) {
      this._lastMsgId = lastMsgId
      return
    }

    if (lastMsgId && lastMsgId != this._lastMsgId) {
      this.notify(state.roomName, lastMsgId, {
        icon: '/static/icon.png',
        body: lastMsg.getIn(['sender', 'name']) + ': ' + lastMsg.get('content'),
      })
      this._lastMsgId = lastMsgId
    }
  },

  closeNotification: function() {
    if (this.notification) {
      // when we close a notification, its onclose callback will get called
      // async. displaying a new notification can race with this, causing the
      // new notification to be invalidly forgotten.
      this.notification.onclose = null
      this.notification.close()
      this.resetNotification()
    }
  },

  resetNotification: function() {
    this.notification = null
    clearTimeout(this._closeTimeout)
    this._closeTimeout = null
  },

  notify: function(message, messageId, options) {
    if (this.focus) {
      return
    }

    Heim.setFavicon('/static/favicon-active.png')

    if (!this.state.enabled) {
      return
    }

    this.closeNotification()

    this.notification = new Notification(message, options)
    this.notification.onclick = function() {
      uiwindow.focus()
      actions.focusMessage(messageId)
    }
    this.notification.onclose = this.resetNotification

    this._closeTimeout = setTimeout(this.closeNotification, this.timeout)
  },
})
