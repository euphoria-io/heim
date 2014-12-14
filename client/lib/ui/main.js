var React = require('react')
var Reflux = require('reflux')

var Chat = require('./chat')

module.exports = React.createClass({
  mixins: [
    Reflux.connect(require('../stores/chat')),
  ],

  render: function() {
    return (
      <div>
        <div>connected: {this.state.connected ? 'yep!' : 'nope'}</div>
        <Chat messages={this.state.messages} />
      </div>
    )
  },
})
