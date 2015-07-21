var React = require('react')

var ThreadListItem = require('./thread-list-item')


module.exports = React.createClass({
  displayName: 'ThreadList',

  mixins: [
    require('react-immutable-render-mixin'),
    require('./tree-node-mixin')('thread'),
  ],

  getDefaultProps: function() {
    return {threadNodeId: '__root'}
  },

  render: function() {
    return (
      <div className="thread-list" onScroll={this.props.onScroll}>
        {this.state.threadNode.get('children').toSeq().map((threadId) =>
          <ThreadListItem key={threadId} threadData={this.props.threadData} threadTree={this.props.threadTree} threadNodeId={threadId} tree={this.props.tree} nodeId={threadId} onClick={this.props.onThreadSelect} />
        ).toArray()}
      </div>
    )
  },
})
