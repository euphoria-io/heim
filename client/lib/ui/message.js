var _ = require('lodash')
var React = require('react')
var moment = require('moment')
var autolinker = require('autolinker')


module.exports = {}

module.exports = React.createClass({
  displayName: 'Message',

  mixins: [require('react-immutable-render-mixin')],

  render: function() {
    var message = this.props.message
    var time = moment.unix(message.get('time'))

    return (
      <div className="line">
        <time dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')}>
          {time.format('h:mma')}
        </time>
        <span className="nick" style={{background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 85%)'}}>{message.getIn(['sender', 'name'])}</span>
        <span className="message" dangerouslySetInnerHTML={{
          __html: autolinker.link(_.escape(message.get('content')), {twitter: false, truncate: 40})
        }} />
      </div>
    )
  },
})
