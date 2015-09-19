import React from 'react'

import Tree from '../tree'
import ThreadListItem from './thread-list-item'
import MessageData from '../message-data'
import TreeNodeMixin from './tree-node-mixin'


export default React.createClass({
  displayName: 'ThreadList',

  propTypes: {
    tree: React.PropTypes.instanceOf(Tree).isRequired,
    threadTree: React.PropTypes.instanceOf(Tree).isRequired,
    threadData: React.PropTypes.instanceOf(MessageData),
    onScroll: React.PropTypes.func,
    onThreadSelect: React.PropTypes.func,
  },

  mixins: [
    require('react-immutable-render-mixin'),
    TreeNodeMixin('thread'),
  ],

  getDefaultProps() {
    return {threadNodeId: '__root'}
  },

  render() {
    return (
      <div className="thread-list" onScroll={this.props.onScroll}>
        {this.state.threadNode.get('children').toSeq().map((threadId) =>
          <ThreadListItem key={threadId} threadData={this.props.threadData} threadTree={this.props.threadTree} threadNodeId={threadId} tree={this.props.tree} nodeId={threadId} onClick={this.props.onThreadSelect} />
        )}
      </div>
    )
  },
})
