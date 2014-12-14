var _ = require('lodash')
var React = require('react')

module.exports = {}

module.exports = React.createClass({
  render: function() {
    return (
      <div className="messages">
        {_.map(this.props.messages, function(message, idx) {
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
