const fs = require('fs')  // needs to be a require to work with brfs for now: https://github.com/babel/babelify/issues/81
import _ from 'lodash'
import Reflux from 'reflux'
import Immutable from 'immutable'

import storage from './storage'
import chat from './chat'
import activity from './activity'


const favicons = module.exports.favicons = {
  'active': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/favicon-active.png', 'base64'),
  'highlight': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/favicon-highlight.png', 'base64'),
  'disconnected': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/favicon-disconnected.png', 'base64'),
}

const icons = module.exports.icons = {
  'active': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/icon.png', 'base64'),
  'highlight': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/icon-highlight.png', 'base64'),
}

const storeActions = Reflux.createActions([
  'enablePopups',
  'disablePopups',
  'pausePopupsUntil',
  'setRoomNotificationMode',
  'dismissNotification',
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
    {onActive: activity.becameActive},
    {onInactive: activity.becameInactive},
  ],

  timeout: 3 * 1000,

  seenExpirationTime: 30 * 1000,

  priority: {
    'new-message': 1,
    'new-reply': 2,
    'new-mention': 3,
  },

  init() {
    this.state = {
      popupsEnabled: null,
      popupsSupported: 'Notification' in window,
      popupsPausedUntil: null,
      soundEnabled: false,
      latestNotifications: Immutable.OrderedMap(),
      notifications: Immutable.OrderedMap(),
      newMessageCount: 0,
    }

    this.active = true
    this.joined = false
    this.connected = false
    this.alerts = {}
    this._roomStorage = null

    if (this.state.popupsSupported) {
      this.state.popupsPermission = Notification.permission === 'granted'
    }

    this._messages = null
    this._notified = {}
    this._dismissed = {}
    this._newNotifications = []
    this._queueUpdateNotifications = _.debounce(this._updateNotifications, 0)
    this._inactiveUpdateNotificationsTimeout = null

    window.addEventListener('unload', this.removeAllAlerts)
  },

  onActive() {
    clearTimeout(this._inactiveUpdateNotificationsTimeout)
    this.active = true
    this.removeAllAlerts()
    this._updateFavicon()
    this._updateTitleCount()
  },

  onInactive() {
    this.active = false
    this._inactiveUpdateNotificationsTimeout = setTimeout(this._updateNotifications, this.seenExpirationTime)
  },

  getInitialState() {
    return this.state
  },

  enablePopups() {
    if (this.state.popupsPermission) {
      storage.set('notify', true)
      storage.set('notifyPausedUntil', null)
    } else {
      Notification.requestPermission(this.onPermission)
    }
  },

  disablePopups() {
    storage.set('notify', false)
    storage.set('notifyPausedUntil', null)
  },

  pausePopupsUntil(time) {
    storage.set('notifyPausedUntil', Math.floor(time))
  },

  setRoomNotificationMode(roomName, mode) {
    storage.setRoom(roomName, 'notifyMode', mode)
  },

  onPermission(permission) {
    this.state.popupsPermission = permission === 'granted'
    if (this.state.popupsPermission) {
      storage.set('notify', true)
    }
    this.trigger(this.state)
  },

  storageChange(data) {
    if (!data) {
      return
    }

    const popupsWereEnabled = this._popupsAreEnabled()
    this.state.popupsEnabled = data.notify
    this.state.popupsPausedUntil = data.notifyPausedUntil
    if (popupsWereEnabled && !this._popupsAreEnabled()) {
      this.closeAllPopups()
    }

    this.state.soundEnabled = data.notifySound
    if (this.state.soundEnabled) {
      // preload audio file
      require('../alertSound')
    }

    this._roomStorage = data.room
    this.trigger(this.state)
  },

  chatStateChange(chatState) {
    this.connected = chatState.connected
    this.joined = chatState.joined
    this._updateFavicon()
  },

  messageReceived(message, state) {
    if (message.get('_own')) {
      // dismiss notifications on siblings or parent of sent messages
      const parentId = message.get('parent')
      if (!parentId || parentId === '__root') {
        return
      }
      module.exports.dismissNotification(parentId)
      const parentMessage = state.messages.get(parentId)
      Immutable.Seq(parentMessage.get('children'))
        .forEach(id => module.exports.dismissNotification(id))
    }
  },

  messagesChanged(ids, state) {
    this._messages = state.messages
    const unseen = Immutable.Seq(ids)
      .map(id => {
        const msg = state.messages.get(id)

        if (id === '__root' || this.state.latestNotifications.has(id)) {
          this._queueUpdateNotifications()
        }

        // exclude already notified
        if (_.has(this._notified, id)) {
          return false
        }

        if (!this._shouldShowNotification(msg, Date.now())) {
          return false
        }

        if (msg.get('_own')) {
          return false
        }

        return msg
      })
      .filter(Boolean)
      .cacheResult()

    unseen.forEach(msg => this._markNotification('new-message', state.roomName, msg))

    unseen
      .filter(msg => {
        const msgId = msg.get('id')
        const parentId = msg.get('parent')

        if (parentId === '__root') {
          return false
        }

        const parentMessage = state.messages.get(parentId)
        const children = parentMessage.get('children').toList()

        if (parentMessage.get('_own') && children.first() === msgId) {
          return true
        }

        const prevChild = children.get(children.indexOf(msgId) - 1)
        return prevChild && state.messages.get(prevChild).get('_own')
      })
      .forEach(msg => this._markNotification('new-reply', state.roomName, msg))

    unseen
      .filter(msg => msg.get('_mention'))
      .forEach(msg => this._markNotification('new-mention', state.roomName, msg))
  },

  _markNotification(kind, roomName, message) {
    this._newNotifications.push({
      kind: kind,
      roomName: roomName,
      message: message,
    })
    this._queueUpdateNotifications()
  },

  _shouldShowNotification(msg, now) {
    if (!msg) {
      return false
    }

    if (_.has(this._dismissed, msg.get('id'))) {
      return false
    }

    if (msg.get('deleted') || !msg.has('$count')) {
      return false
    }

    const seen = msg.get('_seen')
    if (seen === true || seen && seen <= now - this.seenExpirationTime) {
      return false
    }

    return true
  },

  _updateNotifications() {
    const now = Date.now()
    let alerts = {}

    const groups = this.state.latestNotifications
      .withMutations(notifications => {
        _.each(this._newNotifications, newNotification => {
          const newMessageId = newNotification.message.get('id')
          const existingNotification = notifications.get(newMessageId)
          const newPriority = this.priority[newNotification.kind]
          const oldPriority = existingNotification && this.priority[existingNotification.kind] || 0
          if (newPriority > oldPriority) {
            notifications.set(newMessageId, newNotification)
            alerts[newNotification.kind] = newNotification
            if (!this.active && !this._notified[newMessageId]) {
              this.state.newMessageCount++
            }
            this._notified[newMessageId] = true
          }
        })
      })
      .sortBy(notification => notification.message.get('time'))
      .groupBy(notification => notification.kind)

    const newMention = alerts['new-mention']
    if (newMention) {
      const newMentionId = newMention.message.get('id')
      alerts = _.reject(alerts, a => a.message.get('id') === newMentionId)
      this._notifyAlert('new-mention', newMention.roomName, newMention.message, {
        favicon: favicons.highlight,
        icon: icons.highlight,
      })
    }

    const newMessage = alerts['new-reply'] || alerts['new-message']
    if (newMessage) {
      this._notifyAlert(alerts['new-reply'] ? 'new-reply' : 'new-message', newMessage.roomName, newMessage.message, {
        favicon: favicons.active,
        icon: icons.active,
        timeout: this.timeout,
      })
    }

    this._updateFavicon()
    this._updateTitleCount()

    const empty = Immutable.OrderedMap()

    this.state.latestNotifications = empty.concat(
      groups.get('new-mention', empty),
      groups.get('new-reply', empty).takeLast(6),
      groups.get('new-message', empty).takeLast(3)
    )

    this.state.notifications = this.state.latestNotifications
      .filterNot((notification, id) => {
        if (!this._shouldShowNotification(this._messages.get(id), now)) {
          if (this.state.notifications.has(id)) {
            this.removeAlert(notification.kind, id)
          }
          return true
        }
      })
      .map(notification => notification.kind)

    this._newNotifications = []

    this.trigger(this.state)
  },

  _updateFavicon() {
    if (!this.connected) {
      Heim.setFavicon(favicons.disconnected)
    } else {
      const alert = this.alerts['new-mention'] || this.alerts['new-message']
      Heim.setFavicon(alert ? alert.favicon : '/static/favicon.png')
    }
  },

  _updateTitleCount() {
    Heim.setTitleMsg(this.state.newMessageCount || '')
  },

  dismissNotification(messageId) {
    const kind = this.state.notifications.get(messageId)
    if (kind) {
      this.removeAlert(kind, messageId)
      this._dismissed[messageId] = true
      this._queueUpdateNotifications()
      this.trigger(this.state)
    }
  },

  closePopup(kind) {
    const alert = this.alerts[kind]
    if (!alert) {
      return
    }
    clearTimeout(this.alerts[kind].timeout)
    // when we close a notification, its onclose callback will get called
    // async. displaying a new notification can race with this, causing the
    // new notification to be invalidly forgotten.
    if (alert.popup) {
      alert.popup.onclose = () => {
        alert.popup = null
      }
      alert.popup.close()

      // hack: sometimes chrome doesn't close notififcations when we tell it to
      // (while still animating showing it, perhaps?). this failsafe seems to
      // do the trick.
      setTimeout(() => {
        if (alert.popup !== null) {
          alert.popup.close()
        }
      }, 500)
    }
  },

  closeAllPopups() {
    _.each(this.alerts, (alert, kind) => this.closePopup(kind))
  },

  removeAlert(kind, messageId) {
    const alert = this.alerts[kind]
    if (!alert) {
      return
    }
    if (messageId && this.alerts[kind].messageId !== messageId) {
      return
    }
    this.closePopup(kind)
    delete this.alerts[kind]
  },

  removeAllAlerts() {
    _.each(this.alerts, (alert, kind) => this.removeAlert(kind))
    this.state.newMessageCount = 0
  },

  _popupsAreEnabled() {
    if (!this.state.popupsEnabled) {
      return false
    }

    const popupsPaused = this.state.popupsPausedUntil && Date.now() < this.state.popupsPausedUntil
    return !popupsPaused
  },

  _shouldPopup(kind, roomName) {
    const priority = this.priority[kind]
    const notifyMode = _.get(this._roomStorage, [roomName, 'notifyMode'], 'mention')
    if (notifyMode === 'none') {
      return false
    } else if (priority < this.priority['new-' + notifyMode]) {
      return false
    }
    return true
  },

  _notifyAlert(kind, roomName, message, options) {
    if (this.active) {
      return
    }

    const shouldPopup = this._shouldPopup(kind, roomName)

    // note: alert state encompasses favicon state too, so we need to replace
    // the alert regardless of whether we're configured to show a popup

    let alertKind = kind
    if (kind === 'new-reply') {
      // have new reply notifications replace new messages and vice versa
      alertKind = 'new-message'
    }

    this.removeAlert(alertKind)

    const messageId = message.get('id')
    const alert = this.alerts[alertKind] = {}
    alert.messageId = messageId
    alert.favicon = options.favicon
    delete options.favicon

    if (!this.joined || !shouldPopup) {
      return
    }

    if (this.state.popupsPermission && this._popupsAreEnabled()) {
      if (this.state.soundEnabled && alertKind === 'new-mention') {
        require('../alertSound').play()
      }

      const timeoutDuration = options.timeout
      delete options.timeout

      options.body = message.getIn(['sender', 'name']) + ': ' + message.get('content')

      try {
        alert.popup = new Notification(roomName, options)
      } catch (err) {
        Raven.captureException(err)
        return
      }

      const ui = require('./ui')  // avoid import loop
      alert.popup.onclick = () => {
        uiwindow.focus()
        this.dismissNotification(messageId)
        ui.gotoMessageInPane(messageId)
      }
      alert.popup.onclose = _.partial(this.closePopup, alertKind)

      if (timeoutDuration) {
        alert.timeout = setTimeout(alert.popup.onclose, timeoutDuration)
      }
    }
  },
})
