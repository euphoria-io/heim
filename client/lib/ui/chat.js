var _ = require('lodash')
var React = require('react')
var moment = require('moment')

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
      <div ref="messages" className="messages">
        {_.map(this.props.messages, function(message, idx) {
          var time = moment.unix(message.time)

          return (
            <div key={idx} className="line">
              <time dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')}>
                {time.format('h:mma')}
              </time>
              <span className="name">{message.sender}</span>
              <span className="message">{message.content}</span>
            </div>
          )
        })}
      </div>
    )
  },
})
