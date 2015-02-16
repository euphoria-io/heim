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
    this._lastMsgId = null

    if (this.state.supported) {
      this.state.permission = Notification.permission == 'granted'
    }
  },

  focusChange: function(state) {
    this.focus = state.windowFocused
    if (this.focus) {
      this.setFavicon('/static/favicon.png')
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
    var lastMsg = state.messages.last()
    if (!lastMsg) {
      return
    }

    var lastMsgId = lastMsg.get('id')
    if (lastMsgId == this._lastMsgId) {
      return
    }
    this._lastMsgId = lastMsgId
    this.notify(state.roomName, lastMsgId, {
      icon: '/static/icon.png',
      body: lastMsg.getIn(['sender', 'name']) + ': ' + lastMsg.get('content'),
    })
  },

  closeNotification: function() {
    if (this.notification) {
      this.notification.close()
      this.resetNotification()
    }
  },

  resetNotification: function() {
    this.notification = null
    if (this._closeTimeout) {
      clearTimeout(this._closeTimeout)
    }
    this._closeTimeout = null
  },

  notify: function(message, messageId, options) {
    if (this.focus) {
      return
    }

    this.setFavicon('/static/favicon-active.png')

    if (!this.state.enabled || this.notification) {
      return
    }

    this.resetNotification()

    this.notification = new Notification(message, options)
    this.notification.onclick = function() {
      window.focus()
      actions.focusMessage(messageId)
    }
    this.notification.onclose = this.resetNotification

    this._closeTimeout = setTimeout(this.closeNotification, this.timeout)
  },

  setFavicon: require('favicon-setter'),
})
