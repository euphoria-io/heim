import React from 'react'
import classNames from 'classnames'
import Reflux from 'reflux'

import { Pane } from '../stores/ui'

export default React.createClass({
  displayName: 'EntryDragHandle',

  propTypes: {
    pane: React.PropTypes.instanceOf(Pane).isRequired,
  },

  mixins: [
    Reflux.ListenerMixin,
  ],

  getInitialState() {
    return {
      pane: this.props.pane.store.getInitialState(),
    }
  },

  componentDidMount() {
    this.listenTo(this.props.pane.store, state => this.setState({'pane': state}))
  },

  render() {
    const showJumpToBottom = this.state.pane.draggingEntry && this.state.pane.focusedMessage !== null
    return (
      <div className="drag-handle-container">
        <button className={classNames('drag-handle', {'touching': this.state.pane.draggingEntry})} />
        {showJumpToBottom && <button className={classNames('jump-to-bottom', {'touching': this.state.pane.draggingEntryCommand === 'to-bottom'})} />}
      </div>
    )
  },
})
