var React = require('react')
var Reflux = require('reflux')

var actions = require('../actions')


module.exports = React.createClass({
  displayName: 'NickEntry',

  mixins: [
    React.addons.LinkedStateMixin,
    require('./entry-mixin'),
    Reflux.ListenerMixin,
    Reflux.connect(require('../stores/chat').store, 'chat'),
  ],

  componentDidMount: function() {
    this.listenTo(this.props.pane.focusEntry, 'focus')
    this.listenTo(this.props.pane.blurEntry, 'blur')
    this.listenTo(this.props.pane.keydownOnPane, 'proxyKeyDown')
  },

  getInitialState: function() {
    return {value: ''}
  },

  setNick: function(ev) {
    this.refs.input.getDOMNode().focus()
    ev.preventDefault()

    actions.setNick(this.state.value)
  },

  render: function() {
    return (
      <div className="entry-box welcome">
        <div className="message">
          <h1><strong>Hello{this.state.value ? ' ' + this.state.value : ''}!</strong> <span className="no-break">Welcome to our discussion.</span></h1>
          <p>To reply to a message directly, {Heim.isTouch ? 'tap' : 'use the arrow keys or click on'} it.</p>
        </div>
        <form className="entry" onSubmit={this.setNick}>
          <label>choose your name to begin:</label>
          <input key="nick" ref="input" type="text" autoFocus valueLink={this.linkState('value')} disabled={this.state.chat.connected === false} />
        </form>
      </div>
    )
  },
})
