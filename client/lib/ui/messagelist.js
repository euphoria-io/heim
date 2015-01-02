var React = require('react/addons')

var Message = require('./message')


module.exports = React.createClass({
  displayName: 'MessageList',

  mixins: [
    require('react-immutable-render-mixin'),
    require('./treenodemixin'),
  ],

  getDefaultProps: function() {
    return {nodeId: '__root', depth: 0}
  },

  render: function() {
    return (
      <div className="message-list">
        {this.state.node.get('children').toSeq().map(function(nodeId) {
          return <Message key={nodeId} tree={this.props.tree} nodeId={nodeId} depth={this.props.depth} />
        }, this).toArray()}
      </div>
    )
  },
})
