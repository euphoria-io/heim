var React = require('react/addons')

var Message = require('./message')


module.exports = React.createClass({
  displayName: 'MessageList',

  mixins: [
    require('react-immutable-render-mixin'),
    require('./tree-node-mixin')(),
  ],

  getDefaultProps: function() {
    return {nodeId: '__root', depth: 0}
  },

  render: function() {
    var children = this.state.node.get('children')
    return (
      <div className="message-list">
        {children.toIndexedSeq().map((nodeId, idx) =>
          <Message key={nodeId} pane={this.props.pane} tree={this.props.tree} nodeId={nodeId} showTimeAgo={idx == children.size - 1} showTimeStamps={this.props.showTimeStamps} roomSettings={this.props.roomSettings} />
        ).toArray()}
      </div>
    )
  },

  componentDidMount: function() {
    this.props.pane.messageRenderFinished()
  },

  componentDidUpdate: function() {
    this.props.pane.messageRenderFinished()
  },
})
