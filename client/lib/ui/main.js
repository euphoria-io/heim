var React = require('react')
var Reflux = require('reflux')

var actions = require('../actions')
var Chat = require('./chat')

module.exports = React.createClass({
  mixins: [
    Reflux.connect(require('../stores/chat')),
  ],

  send: function(ev) {
    var input = this.refs.line.getDOMNode()
    actions.sendMessage(input.value)
    input.value = ''
    ev.preventDefault()
  },

  render: function() {
    return (
      <div>
        <div>connected: {this.state.connected ? 'yep!' : 'nope'}</div>
        <Chat messages={this.state.messages} />
        <form onSubmit={this.send}>
          <input ref="line" type="text" />
        </form>
      </div>
    )
  },
})
