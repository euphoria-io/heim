var _ = require('lodash')
var React = require('react')

module.exports = {}

module.exports = React.createClass({
  componentWillUpdate: function() {
    // via http://blog.vjeux.com/2013/javascript/scroll-position-with-react.html
    var node = this.refs.messages.getDOMNode()
    this._atBottom = node.scrollTop + node.offsetHeight >= node.scrollHeight
  },

  componentDidUpdate: function() {
    if (this._atBottom) {
      var node = this.refs.messages.getDOMNode()
      node.scrollTop = node.scrollHeight
    }
  },

  render: function() {
    return (
      <div ref="messages" className="messages" onClick={this.props.onClick}>
        {_.map(this.props.messages, function(message, idx) {
          return (
            <div key={idx}>
              {message.content}
            </div>
          )
        })}
      </div>
    )
  },
})
