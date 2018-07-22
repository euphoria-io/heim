import _ from 'lodash'
import React from 'react'
import ReactDOM from 'react-dom'
import classNames from 'classnames'
import Reflux from 'reflux'

import FastButton from './FastButton'
import MessageText from './MessageText'
import LiveTimeAgo from './LiveTimeAgo'
import TreeNodeMixin from './TreeNodeMixin'


export default React.createClass({
  displayName: 'NotificationListItem',

  propTypes: {
    nodeId: React.PropTypes.string.isRequired,
    kind: React.PropTypes.string.isRequired,
    onClick: React.PropTypes.func,
  },

  mixins: [
    TreeNodeMixin(),
    Reflux.connect(require('../stores/clock').minute, 'now'),
  ],

  componentWillEnter(callback) {
    const node = ReactDOM.findDOMNode(this)
    const height = node.clientHeight
    node.style.transition = node.style.webkitTransition = 'none'
    node.style.height = 0
    node.style.opacity = 0
    _.identity(node.offsetHeight)  // reflow so transition starts
    node.style.transition = node.style.webkitTransition = 'all .25s ease'
    node.style.height = height + 'px'
    node.style.opacity = 1
    callback()
  },

  componentWillLeave(callback) {
    const node = ReactDOM.findDOMNode(this)
    node.style.transition = node.style.webkitTransition = 'all .25s ease'
    node.style.height = 0
    setTimeout(() => {
      node.style.transition = 'none'
      callback()
    }, 250)
  },

  render() {
    const message = this.state.node

    return (
      <FastButton component="div" className={classNames('notification', this.props.kind, {'seen': message.get('_seen')})} onClick={ev => this.props.onClick(ev, this.props.nodeId)}>
        <MessageText className="title" content={message.get('content').trim()} maxLength={140} />
        <LiveTimeAgo className="ago" time={message.get('time')} nowText="active" />
      </FastButton>
    )
  },
})
