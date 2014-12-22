var _ = require('lodash')
var Reflux = require('reflux')

var storage = require('./storage')


var actions = Reflux.createActions([
  'enable',
  'disable',
])
_.extend(module.exports, actions)

actions.enable.sync = true

module.exports.store = Reflux.createStore({
  listenables: [
    actions,
    {storageChange: storage.store},
    {chatUpdate: require('./chat').store},
  ],

  init: function() {
    this.state = {
      enabled: null,
      supported: 'Notification' in window
    }

    this.focus = true
    this.notification = null

    if (this.state.supported) {
      this.state.permission = Notification.permission == 'granted'
    }

    window.addEventListener('focus', this.onFocus.bind(this), false)
    window.addEventListener('blur', this.onBlur.bind(this), false)
  },

  onFocus: function() {
    this.focus = true
    this.closeNotification()
  },

  onBlur: function() {
    this.focus = false
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
    if (permission == "granted") {
      this.state.permission = true
      storage.set('notify', true)
    }
  },

  storageChange: function(data) {
    this.state.enabled = this.state.permission && data.notify
    this.trigger(this.state)
  },

  chatUpdate: function(state) {
    var lastMsg = state.messages.last()
    if (lastMsg == this._lastMsg) {
      return
    }
    this._lastMsg = lastMsg
    this.notify('new message', {body: lastMsg.getIn(['sender', 'name']) + ': ' + lastMsg.get('content')})
  },

  closeNotification: function() {
    if (this.notification) {
      this.notification.close()
      this.notification = null
    }
  },

  notify: function(message, options) {
    if (this.focus || !this.state.enabled || this.notification) {
      return
    }

    this.notification = new Notification(message, options)
    this.notification.onclick = function() {
      window.focus()
    }
  },
})
