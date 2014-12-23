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
    var hour = time.hour()
    var bgLightness = (hour > 12 ? 24 - hour : hour) / 12
    var timeStyle = {
      background: 'hsla(0, 0%, ' + (100 * bgLightness).toFixed(2) + '%, .175)',
      color: 'hsla(0, 0%, 100%, ' + (.3 + .2 * bgLightness).toFixed(2) + ')',
    }

    return (
      <div className="line">
        <time dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')} style={timeStyle}>
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
