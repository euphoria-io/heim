var _ = require('lodash')
var React = require('react')
var moment = require('moment')
var Autolinker = require('autolinker')

var actions = require('../actions')
var ChatEntry = require('./chatentry')


var autolinker = new Autolinker({
  twitter: false,
  truncate: 40,
  replaceFn: function(autolinker, match) {
    if (match.getType() == 'url') {
      var url = match.getUrl()
      var tag = autolinker.getTagBuilder().build(match)

      if (location.protocol == 'https:' && RegExp('^https?:\/\/' + location.hostname).test(url)) {
        // self-link securely
        tag.setAttr('href', url.replace(/^http:/, 'https:'))
      } else {
        tag.setAttr('rel', 'noreferrer')
      }

      return tag
    }
  },
})

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

    var content = message.get('content')
    var messageRender
    if (/^\/me/.test(content)) {
      content = content.replace(/^\/me ?/, '')
      messageRender = (
        <span className="message message-emote" style={{background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 95%)'}} dangerouslySetInnerHTML={{
          __html: autolinker.link(_.escape(content))
        }} />
      )
    } else {
      messageRender = <span className="message" dangerouslySetInnerHTML={{
        __html: autolinker.link(_.escape(content))
      }} />
    }

    return (
      <div data-message-id={message.get('id')} className="message-node">
        <div className="line" onClick={this.focusMessage}>
          <time dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')} style={timeStyle}>
            {time.format('h:mma')}
          </time>
          <span className="nick" style={{background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 85%)'}}>{message.getIn(['sender', 'name'])}</span>
          {messageRender}
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
