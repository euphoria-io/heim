var _ = require('lodash')
var React = require('react')
var moment = require('moment')


module.exports = {}

module.exports = React.createClass({
  render: function() {
    var now = moment()
    return (
      <div className="messages">
        {_.map(this.props.messages, function(message, idx) {
          var time = moment.unix(message.time)

          return (
            <div key={idx} className="line">
              <time dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')}>
                {time.format('h:mma')}
              </time>
              <span className="nick">{message.sender.name}</span>
              <span className="message">{message.content}</span>
            </div>
          )
        })}
        {this.props.disconnected ?
          <div key="status" className="line status disconnected">
            <time dateTime={now.toISOString()} title={now.format('MMMM Do YYYY, h:mm:ss a')}>
              {now.format('h:mma')}
            </time>
            <span className="message">disconnected!</span>
          </div>
        : null}
      </div>
    )
  },
})
