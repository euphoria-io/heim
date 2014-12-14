var React = require('react')
var Reflux = require('reflux')

var actions = require('../actions')
var Chat = require('./chat')

module.exports = React.createClass({
  mixins: [
    Reflux.connect(require('../stores/chat')),
  ],

  send: function(ev) {
    actions.send(this.refs.line.getDOMNode().value)
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
