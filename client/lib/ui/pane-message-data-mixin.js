module.exports = {
  getInitialState: function() {
    return {paneData: this.props.pane.store.getMessageData(this.props.nodeId)}
  },

  componentWillMount: function() {
    this.props.pane.store.changes.on(this.props.nodeId, this.onDataUpdate)
  },

  componentWillUnmount: function() {
    this.props.pane.store.changes.off(this.props.nodeId, this.onDataUpdate)
  },

  onDataUpdate: function(newValue) {
    this.setState({paneData: newValue})
  },
}
