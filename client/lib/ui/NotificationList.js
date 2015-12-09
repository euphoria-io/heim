import React from 'react'
import ReactTransitionGroup from 'react-addons-transition-group'
import Immutable from 'immutable'

import NotificationListItem from './NotificationListItem'
import Tree from '../tree'


export default React.createClass({
  displayName: 'NotificationList',

  propTypes: {
    tree: React.PropTypes.instanceOf(Tree).isRequired,
    notifications: React.PropTypes.instanceOf(Immutable.OrderedMap).isRequired,
    onNotificationSelect: React.PropTypes.func,
    animate: React.PropTypes.bool,
  },

  render() {
    const notifications = this.props.notifications.map((kind, messageId) =>
      <NotificationListItem key={messageId} tree={this.props.tree} nodeId={messageId} kind={kind} onClick={this.props.onNotificationSelect} />
    ).toIndexedSeq()

    if (this.props.animate) {
      return (
        <ReactTransitionGroup component="div" className="notification-list">
          {notifications}
        </ReactTransitionGroup>
      )
    }
    return (
      <div className="notification-list">
        {notifications}
      </div>
    )
  },
})
