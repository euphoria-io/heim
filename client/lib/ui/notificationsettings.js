var React = require('react/addons')
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

  render: function() {
    if (!this.state.notification.popupsSupported) {
      return <span className="notification-settings" />
    }

    var notificationsButton
    if (!this.state.notification.popupsPermission) {
      notificationsButton = <FastButton className="notifications " onClick={this.enableNotify}>enable notifications</FastButton>
    } else {
      if (this.state.notification.popupsPausedUntil && Date.now() < this.state.notification.popupsPausedUntil) {
        notificationsButton = <FastButton className="notifications snoozed" onClick={this.disableNotify}>{'for ' + this.state.pauseTimeRemaining}</FastButton>
      } else if (!this.state.notification.popupsEnabled) {
        notificationsButton = <FastButton className="notifications paused" onClick={this.enableNotify}>notifications</FastButton>
      } else {
        notificationsButton = <FastButton className="notifications normal" onClick={this.snoozeNotify}>notifications</FastButton>
      }
    }

    return  (
      <span className="notification-settings">
        {notificationsButton}
      </span>
    )
  },
})
