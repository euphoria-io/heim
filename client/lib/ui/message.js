var _ = require('lodash')
var React = require('react')
var moment = require('moment')
var autolinker = require('autolinker')

var actions = require('../actions')
var ChatEntry = require('./chatentry')


var Message = module.exports = React.createClass({
  displayName: 'Message',

  mixins: [
    require('react-immutable-render-mixin'),
    require('./treenodemixin'),
  ],

  focusMessage: function() {
    actions.toggleFocusMessage(this.props.nodeId, this.state.node.get('parent'))
  },

  render: function() {
    var message = this.state.node
    var children = message.get('children')
    var entry = message.get('entry')
    var time = moment.unix(message.get('time'))
    var hour = time.hour() + time.minute() / 60
    var bgLightness = (hour > 12 ? 24 - hour : hour) / 12
    var timeStyle = {
      background: 'hsla(0, 0%, ' + (100 * bgLightness).toFixed(2) + '%, .175)',
      color: 'hsla(0, 0%, 100%, ' + (0.3 + 0.2 * bgLightness).toFixed(2) + ')',
      // kludge timestamp columns into not indenting along with thread
      marginLeft: -this.props.depth * 10,
    }

    return (
      <div data-message-id={message.get('id')} className="message-node">
        <div className="line" onClick={this.focusMessage}>
          <time dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')} style={timeStyle}>
            {time.format('h:mma')}
          </time>
          <span className="nick" style={{background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 85%)'}}>{message.getIn(['sender', 'name'])}</span>
          <span className="message" dangerouslySetInnerHTML={{
            __html: autolinker.link(_.escape(message.get('content')), {twitter: false, truncate: 40})
          }} />
        </div>
        {(children.size > 0 || entry) &&
          <div className="replies">
            {children.toSeq().map(function(nodeId) {
              return <Message key={nodeId} tree={this.props.tree} nodeId={nodeId} depth={this.props.depth + 1} />
            }, this).toArray()}
            {entry && <ChatEntry />}
          </div>
        }
        <div className="reply-anchor" />
      </div>
    )
  },
})
