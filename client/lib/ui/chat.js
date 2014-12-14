var _ = require('lodash')
var React = require('react')
var Reflux = require('reflux')

var ChatStore = require('../stores/chat')

module.exports = {}

module.exports = React.createClass({
  mixins: [
    Reflux.connect(ChatStore),
  ],

  getInitialState: function() {
    return {messages: []}
  },

  render: function() {
    return (
      <div className="messages">
        {_.map(this.state.messages, function(message, idx) {
          return (
            <div key={idx}>
              {message.text}
            </div>
          )
        })}
      </div>
    )
  },
})
