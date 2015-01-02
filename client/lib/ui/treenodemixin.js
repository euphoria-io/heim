module.exports = {
  getInitialState: function() {
    return {node: this.props.tree.get(this.props.nodeId)}
  },

  componentWillMount: function() {
    this.props.tree.changes.on(this.props.nodeId, this.onNodeUpdate)
  },

  componentWillUnmount: function() {
    this.props.tree.changes.off(this.props.nodeId, this.onNodeUpdate)
  },

  onNodeUpdate: function(newValue) {
    this.setState({node: newValue})
  },
}
