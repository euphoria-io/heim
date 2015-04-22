var React = require('react')
var classNames = require('classnames')
var Reflux = require('reflux')
var moment = require('moment')

var notification = require('../stores/notification')
var storage = require('../stores/storage')
var clock = require('../stores/clock')
var FastButton = require('./fast-button')


module.exports = React.createClass({
  displayName: 'NotificationSettings',

  mixins: [
    Reflux.connect(notification.store, 'notification'),
    Reflux.connect(storage.store, 'storage'),
    Reflux.connect(clock.second, 'now'),
  ],

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
        var pauseTimeRemaining = moment(this.state.notification.popupsPausedUntil).from(this.state.now, true)
        notificationsClass = 'snoozed'
        notificationsButton = <FastButton className="notification-toggle" onClick={this.disableNotify} title="pause notifications">{'for ' + pauseTimeRemaining}</FastButton>
      } else if (!this.state.notification.popupsEnabled) {
        notificationsClass = 'paused'
        notificationsButton = <FastButton className="notification-toggle" onClick={this.enableNotify} title="resume notifications">for now</FastButton>
      } else {
        notificationsClass = 'enabled'
        notificationsButton = <FastButton className="notification-toggle" onClick={this.snoozeNotify} title="snooze notifications">notifications</FastButton>
      }

      var roomStorage = this.state.storage.room[this.props.roomName] || {}
      var currentMode = roomStorage.notifyMode || 'mention'
      notificationModeUI = (
        <span className="mode-selector">
          <FastButton className={classNames({'mode': true, 'none': true, 'selected': currentMode == 'none'})} onClick={() => this.setMode('none')} title="no notifications">none</FastButton>
          <FastButton className={classNames({'mode': true, 'mention': true, 'selected': currentMode == 'mention'})} onClick={() => this.setMode('mention')} title="only notify @mentions">mention</FastButton>
          <FastButton className={classNames({'mode': true, 'reply': true, 'selected': currentMode == 'reply'})} onClick={() => this.setMode('reply')} title="notify @mentions and replies to your messages">reply</FastButton>
          <FastButton className={classNames({'mode': true, 'message': true, 'selected': currentMode == 'message'})} onClick={() => this.setMode('message')} title="notify all messages">message</FastButton>
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
