var React = require('react')
var Reflux = require('reflux')

var actions = require('../actions')
var Chat = require('./chat')

module.exports = React.createClass({
  mixins: [
    Reflux.connect(require('../stores/chat')),
  ],

  send: function(ev) {
    var input = this.refs.input.getDOMNode()
    actions.sendMessage(input.value)
    input.value = ''
    ev.preventDefault()
  },

  focusInput: function() {
    this.refs.input.getDOMNode().focus()
  },

  render: function() {
    return (
      <div>
        <div>connected: {this.state.connected ? 'yep!' : 'nope'}</div>
        <Chat messages={this.state.messages} onClick={this.focusInput} />
        <form onSubmit={this.send}>
          <input ref="input" type="text" autoFocus />
        </form>
      </div>
    )
  },
})
