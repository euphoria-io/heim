var React = require('react')
var ReactTransitionGroup = React.addons.TransitionGroup

var NotificationListItem = require('./notification-list-item')


module.exports = React.createClass({
  displayName: 'NotificationList',

  render: function() {
    var notifications = this.props.notifications.map((kind, messageId) =>
      <NotificationListItem key={messageId} tree={this.props.tree} nodeId={messageId} kind={kind} onClick={this.props.onNotificationSelect} />
    ).toArray()

    if (this.props.animate) {
      return  (
        <ReactTransitionGroup component="div" className="notification-list">
          {notifications}
        </ReactTransitionGroup>
      )
    } else {
      return  (
        <div className="notification-list">
          {notifications}
        </div>
      )
    }
  },
})
