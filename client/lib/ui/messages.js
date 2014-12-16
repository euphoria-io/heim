var _ = require('lodash')
var React = require('react')
var moment = require('moment')

var Message = require('./message')


module.exports = {}

module.exports = React.createClass({
  displayName: 'Messages',

  mixins: [require('react-immutable-render-mixin')],

  render: function() {
    var now = moment()
    return (
      <div className="messages">
        {this.props.messages.map(function(message, idx) {
          return <Message key={idx} message={message} />
        }, this).toArray()}
        {this.props.disconnected ?
          <div key="status" className="line status disconnected">
            <time dateTime={now.toISOString()} title={now.format('MMMM Do YYYY, h:mm:ss a')}>
              {now.format('h:mma')}
            </time>
            <span className="message">reconnecting...</span>
          </div>
        : null}
      </div>
    )
  },
})
