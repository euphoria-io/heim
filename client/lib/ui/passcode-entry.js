import _ from 'lodash'
import React from 'react'
import Reflux from 'reflux'

import actions from '../actions'
import chat from '../stores/chat'
import { Pane } from '../stores/ui'
import EntryMixin from './entry-mixin'


export default React.createClass({
  displayName: 'PasscodeEntry',

  propTypes: {
    pane: React.PropTypes.instanceOf(Pane).isRequired,
  },

  mixins: [
    React.addons.LinkedStateMixin,
    EntryMixin,
    Reflux.listenTo(chat.store, '_onChatUpdate'),
  ],

  getInitialState() {
    return {
      value: '',
      connected: null,
      authState: null,
    }
  },

  componentWillMount() {
    // debounce state changes to reduce jank from fast responses
    // TODO: break out into a debounced connect mixin, once chat store is fully immutable?
    this._onChatUpdate = _.debounce(this.onChatUpdate, 250, {leading: true, trailing: true})
  },

  componentDidMount() {
    this.listenTo(this.props.pane.focusEntry, 'focus')
    this.listenTo(this.props.pane.blurEntry, 'blur')
    this.listenTo(this.props.pane.keydownOnPane, 'proxyKeyDown')
  },

  onChatUpdate(chatState) {
    this.setState({
      connected: chatState.connected,
      authState: chatState.authState,
    })
  },

  tryPasscode(ev) {
    this.refs.input.getDOMNode().focus()
    ev.preventDefault()

    if (this.state.authState === 'trying') {
      return
    }

    actions.tryRoomPasscode(this.state.value)
    this.setState({value: ''})
  },

  render() {
    let label
    switch (this.authState) {
    case 'trying':
      label = 'trying...'
      break
    case 'failed':
      label = 'no dice. try again:'
      break
    default:
      label = 'passcode:'
    }

    return (
      <div className="entry-box passcode">
        <p className="message">This room requires a passcode.</p>
        <form className="entry focus-target" onSubmit={this.tryPasscode}>
          <label>{label}</label>
          <input key="passcode" ref="input" type="password" className="entry-text" autoFocus valueLink={this.linkState('value')} disabled={this.state.connected === false} />
        </form>
      </div>
    )
  },
})
