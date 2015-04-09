var React = require('react/addons')
var cx = React.addons.classSet
var Reflux = require('reflux')
var moment = require('moment')

var notification = require('../stores/notification')
var storage = require('../stores/storage')
var FastButton = require('./fastbutton')


module.exports = React.createClass({
  displayName: 'Settings',

  mixins: [
    Reflux.listenTo(notification.store, 'notificationChange', 'notificationChange'),
    Reflux.connect(notification.store, 'notification'),
    Reflux.connect(storage.store, 'storage'),
  ],

  notificationChange: function(state) {
    if (state.popupsPausedUntil) {
      if (!this._updateInterval) {
        this._updateInterval = setInterval(this.updateTimeRemaining, 5 * 1000)
      }
      this.updateTimeRemaining()
    } else {
      if (this._updateInterval) {
        clearInterval(this._updateInterval)
        this._updateInterval = null
      }
    }
  },

  updateTimeRemaining: function() {
    this.setState({
      pauseTimeRemaining: moment(this.state.notification.popupsPausedUntil).fromNow(true)
    })
  },

  enableNotify: function() {
    notification.enablePopups()
  },

  snoozeNotify: function() {
    notification.pausePopupsUntil(Date.now() + 1 * 60 * 60 * 1000)
  },

  disableNotify: function() {
    notification.disablePopups()
  },

  setMode: function(mode) {
    notification.setRoomNotificationMode(this.props.roomName, mode)
  },

  render: function() {
    if (!this.state.notification.popupsSupported) {
      return <span className="notification-settings" />
    }

    var notificationsClass
    var notificationsButton
    var notificationModeUI
    if (!this.state.notification.popupsPermission) {
      notificationsClass = 'disabled'
      notificationsButton = <FastButton className="notification-toggle" onClick={this.enableNotify}>enable notifications</FastButton>
    } else {
      if (this.state.notification.popupsPausedUntil && Date.now() < this.state.notification.popupsPausedUntil) {
        notificationsClass = 'snoozed'
        notificationsButton = <FastButton className="notification-toggle" onClick={this.disableNotify}>{'for ' + this.state.pauseTimeRemaining}</FastButton>
      } else if (!this.state.notification.popupsEnabled) {
        notificationsClass = 'paused'
        notificationsButton = <FastButton className="notification-toggle" onClick={this.enableNotify}>for now</FastButton>
      } else {
        notificationsClass = 'enabled'
        notificationsButton = <FastButton className="notification-toggle" onClick={this.snoozeNotify}>notifications</FastButton>
      }

      var roomStorage = this.state.storage.room[this.props.roomName] || {}
      var currentMode = roomStorage.notifyMode || 'mention'
      notificationModeUI = (
        <span className="mode-selector">
          <FastButton className={cx({'mode': true, 'none': true, 'selected': currentMode == 'none'})} onClick={() => this.setMode('none')}>none</FastButton>
          <FastButton className={cx({'mode': true, 'mention': true, 'selected': currentMode == 'mention'})} onClick={() => this.setMode('mention')}>mention</FastButton>
          <FastButton className={cx({'mode': true, 'message': true, 'selected': currentMode == 'message'})} onClick={() => this.setMode('message')}>message</FastButton>
        </span>
      )
    }

    return  (
      <span className={'notification-settings ' + notificationsClass}>
        {notificationsButton}
        {notificationModeUI}
      </span>
    )
  },
})
