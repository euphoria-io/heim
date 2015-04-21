var _ = require('lodash')
var React = require('react')
var Reflux = require('reflux')

var actions = require('../actions')
var chat = require('../stores/chat')


module.exports = React.createClass({
  displayName: 'PasscodeEntry',

  mixins: [
    React.addons.LinkedStateMixin,
    require('./entry-mixin'),
    Reflux.listenTo(chat.store, '_onChatUpdate'),
    Reflux.listenTo(actions.focusEntry, 'focus'),
    Reflux.listenTo(actions.keydownOnEntry, 'proxyKeyDown'),
  ],

  componentWillMount: function() {
    // debounce state changes to reduce jank from fast responses
    // TODO: break out into a debounced connect mixin, once chat store is fully immutable?
    this._onChatUpdate = _.debounce(this.onChatUpdate, 250, {leading: true, trailing: true})
  },

  getInitialState: function() {
    return {
      value: '',
      connected: null,
      authState: null,
    }
  },

  onChatUpdate: function(chatState) {
    this.setState({
      connected: chatState.connected,
      authState: chatState.authState,
    })
  },

  tryPasscode: function(ev) {
    this.refs.input.getDOMNode().focus()
    ev.preventDefault()

    if (this.state.authState == 'trying') {
      return
    }

    actions.tryRoomPasscode(this.state.value)
    this.setState({value: ''})
  },

  render: function() {
    return (
      <div className="entry-box passcode">
        <p className="message">This room requires a passcode.</p>
        <form className="entry" onSubmit={this.tryPasscode}>
          <label>{
            this.state.authState == 'trying' ? 'trying...'
              : this.state.authState == 'failed' ? 'no dice. try again:'
                : 'passcode:'
          }</label>
          <input key="passcode" ref="input" type="password" autoFocus valueLink={this.linkState('value')} disabled={this.state.connected === false} />
        </form>
      </div>
    )
  },
})
