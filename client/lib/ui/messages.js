var _ = require('lodash')
var React = require('react')
var moment = require('moment')
var autolinker = require('autolinker')


module.exports = {}

module.exports = React.createClass({
  displayName: 'Messages',

  mixins: [require('react-immutable-render-mixin')],

  render: function() {
    var now = moment()
    return (
      <div className="messages">
        {this.props.messages.map(function(message, idx) {
          var time = moment.unix(message.time)

          return (
            <div key={idx} className="line">
              <time dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')}>
                {time.format('h:mma')}
              </time>
              <span className="nick" style={{background: 'hsl(' + message.sender.hue + ', 65%, 85%)'}}>{message.sender.name}</span>
              <span className="message" dangerouslySetInnerHTML={{
                __html: autolinker.link(_.escape(message.content), {twitter: false, truncate: 40})
              }} />
            </div>
          )
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
