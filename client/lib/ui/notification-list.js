var React = require('react')
var ReactTransitionGroup = React.addons.TransitionGroup

var NotificationListItem = require('./notification-list-item')


module.exports = React.createClass({
  displayName: 'NotificationList',

  render: function() {
    return  (
      <ReactTransitionGroup component="div" className="notification-list">
        {this.props.notifications.map((kind, messageId) =>
          <NotificationListItem key={messageId} tree={this.props.tree} nodeId={messageId} kind={kind} onClick={this.props.onNotificationSelect} />
        ).toArray()}
      </ReactTransitionGroup>
    )
  },
})
