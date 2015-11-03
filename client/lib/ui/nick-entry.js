import React from 'react'
import Reflux from 'reflux'

import actions from '../actions'
import { Pane } from '../stores/ui'
import EntryMixin from './entry-mixin'


export default React.createClass({
  displayName: 'NickEntry',

  propTypes: {
    pane: React.PropTypes.instanceOf(Pane).isRequired,
  },

  mixins: [
    require('react-addons-linked-state-mixin'),
    EntryMixin,
    Reflux.ListenerMixin,
    Reflux.connect(require('../stores/chat').store, 'chat'),
  ],

  getInitialState() {
    return {value: ''}
  },

  componentDidMount() {
    this.listenTo(this.props.pane.focusEntry, 'focus')
    this.listenTo(this.props.pane.blurEntry, 'blur')
    this.listenTo(this.props.pane.keydownOnPane, 'proxyKeyDown')
  },

  setNick(ev) {
    this.refs.input.focus()
    ev.preventDefault()

    actions.setNick(this.state.value)
  },

  render() {
    return (
      <div className="entry-box welcome">
        <div className="message">
          <h1><strong>Hello{this.state.value ? ' ' + this.state.value : ''}!</strong> <span className="no-break">Welcome to our discussion.</span></h1>
          <p>To reply to a message directly, {Heim.isTouch ? 'tap' : 'use the arrow keys or click on'} it.</p>
        </div>
        <form className="entry focus-target" onSubmit={this.setNick}>
          <label>choose your name to begin:</label>
          <input key="nick" ref="input" type="text" className="entry-text" autoFocus valueLink={this.linkState('value')} disabled={this.state.chat.connected === false} />
        </form>
      </div>
    )
  },
})
