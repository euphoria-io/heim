var React = require('react/addons')
var classNames = require('classnames')
var Reflux = require('reflux')

module.exports = React.createClass({
  displayName: 'EntryDragHandle',

  mixins: [
    Reflux.ListenerMixin,
  ],

  componentDidMount: function() {
    this.listenTo(this.props.pane.store, state => this.setState({'pane': state}))
  },

  getInitialState: function() {
    return {
      pane: this.props.pane.store.getInitialState(),
    }
  },

  render: function() {
    var showJumpToBottom = this.state.pane.draggingEntry && this.state.pane.focusedMessage !== null
    return (
      <div className="drag-handle-container">
        <button className={classNames('drag-handle', {'touching': this.state.pane.draggingEntry})} />
        {showJumpToBottom && <button className={classNames('jump-to-bottom', {'touching': this.state.pane.draggingEntryCommand == 'to-bottom'})} />}
      </div>
    )
  },
})
