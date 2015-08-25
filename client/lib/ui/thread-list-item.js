var React = require('react')
var Reflux = require('reflux')
var classNames = require('classnames')

var FastButton = require('./fast-button')
var MessageText = require('./message-text')
var LiveTimeAgo = require('./live-time-ago')


var ThreadListItem = module.exports = React.createClass({
  displayName: 'ThreadListItem',

  mixins: [
    require('react-immutable-render-mixin'),
    require('./tree-node-mixin')('thread'),
    require('./tree-node-mixin')(),
    require('./message-data-mixin')(props => props.threadData, 'threadData'),
    Reflux.connect(require('../stores/clock').minute, 'now'),
  ],

  getDefaultProps: function() {
    return {
      depth: 0,
    }
  },

  render: function() {
    var thread = this.state.threadNode
    var message = this.state.node

    var count = this.props.tree.getCount(this.props.nodeId)
    if (!count) {
      // FIXME: due to react batching when new logs are loaded, this component
      // can update after the node has been cleared (with shadow data) but
      // before being removed.
      return <div />
    }

    var newCount = count.get('newDescendants')
    var children = thread.get('children')
    var timestamp

    if (children.size) {
      var childrenNewCount = children
        .map(childId => this.props.tree.getCount(childId).get('newDescendants'))
        .reduce((a, b) => a + b, 0)
      newCount -= childrenNewCount
      timestamp = this.props.tree.get(message.get('children').last()).get('time')
    } else {
      timestamp = count.get('latestDescendantTime')
    }

    var isActive = this.state.now - timestamp * 1000 < 30 * 60 * 1000

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
          ).toArray()}
        </div>}
      </div>
    )
  },
})
