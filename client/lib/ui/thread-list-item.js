import React from 'react'
import Reflux from 'reflux'
import classNames from 'classnames'

import FastButton from './fast-button'
import MessageText from './message-text'
import LiveTimeAgo from './live-time-ago'
import Tree from '../tree'
import MessageData from '../message-data'
import TreeNodeMixin from './tree-node-mixin'
import MessageDataMixin from './message-data-mixin'


const ThreadListItem = React.createClass({
  displayName: 'ThreadListItem',

  propTypes: {
    nodeId: React.PropTypes.string.isRequired,
    tree: React.PropTypes.instanceOf(Tree).isRequired,
    threadNodeId: React.PropTypes.string.isRequired,
    threadTree: React.PropTypes.instanceOf(Tree).isRequired,
    depth: React.PropTypes.number,
    threadData: React.PropTypes.instanceOf(MessageData),
    onClick: React.PropTypes.func,
  },

  mixins: [
    require('react-immutable-render-mixin'),
    TreeNodeMixin('thread'),
    TreeNodeMixin(),
    MessageDataMixin(props => props.threadData, 'threadData'),
    Reflux.connect(require('../stores/clock').minute, 'now'),
  ],

  getDefaultProps() {
    return {
      depth: 0,
    }
  },

  render() {
    const thread = this.state.threadNode
    const message = this.state.node

    const count = this.props.tree.getCount(this.props.nodeId)
    if (!count) {
      // FIXME: due to react batching when new logs are loaded, this component
      // can update after the node has been cleared (with shadow data) but
      // before being removed.
      return <div />
    }

    let newCount = count.get('newDescendants')
    const children = thread.get('children')
    let timestamp

    if (children.size) {
      const childrenNewCount = children
        .map(childId => this.props.tree.getCount(childId).get('newDescendants'))
        .reduce((a, b) => a + b, 0)
      newCount -= childrenNewCount
      timestamp = this.props.tree.get(message.get('children').last()).get('time')
    } else {
      timestamp = count.get('latestDescendantTime')
    }

    const isActive = this.state.now - timestamp * 1000 < 30 * 60 * 1000

    return (
      <div className="thread">
        <FastButton component="div" data-thread-id={this.props.threadNodeId} className={classNames('info', {'selected': this.state.threadData.get('selected'), 'active': isActive})} onClick={ev => this.props.onClick(ev, this.props.threadNodeId)}>
          <MessageText className="title" content={message.get('content')} maxLength={140} />
          {newCount > 0 && <span className={classNames('new-count', {'new-mention': count.get('newMentionDescendants') > 0})}>{newCount}</span>}
          <LiveTimeAgo className="ago" time={timestamp} nowText="active" />
        </FastButton>
        {this.props.depth < 3 && children.size > 0 && <div className="children">
          {children.toSeq().map((threadId) =>
            <ThreadListItem key={threadId} threadData={this.props.threadData} threadTree={this.props.threadTree} threadNodeId={threadId} tree={this.props.tree} nodeId={threadId} depth={this.props.depth + 1} onClick={this.props.onClick} />
          )}
        </div>}
      </div>
    )
  },
})

export default ThreadListItem
