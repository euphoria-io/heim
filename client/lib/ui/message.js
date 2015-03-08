var _ = require('lodash')
var React = require('react')
var cx = React.addons.classSet
var moment = require('moment')

var actions = require('../actions')
var MessageText = require('./messagetext')
var ChatEntry = require('./chatentry')


var Message = module.exports = React.createClass({
  displayName: 'Message',

  mixins: [
    require('react-immutable-render-mixin'),
    require('./treenodemixin'),
  ],

  focusMessage: function() {
    if (!uiwindow.getSelection().isCollapsed) {
      return
    }

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

    var lineClasses = {
      'line': true,
      'focus-highlight': entry || this.props.displayFocusHighlight,
    }

    var content = message.get('content')

    var messageEmbeds
    var embeds = []
    content = content.replace(/(?:https?:\/\/)?(?:www\.|i\.)?imgur.com\/(\w+)(\.\w+)?(\S*)/g, (match, id, ext, rest, offset, string) => {
      if (rest) {
        return string
      }
      embeds.push(<a key={offset} href={'//imgur.com/' + id} target="_blank"><img src={'//i.imgur.com/' + id + (ext == '.gif' ? '' : 't') + (ext || '.jpg')} /></a>)
      return ''
    })
    if (embeds.length) {
      messageEmbeds = <div className="embeds">{embeds}</div>
    }

    var messageRender
    if (!_.trim(content)) {
      messageRender = null
    } else if (/^\/me/.test(content)) {
      content = content.replace(/^\/me ?/, '')
      messageRender = <MessageText content={content} className="message message-emote" style={{background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 95%)'}} />
      lineClasses['line-emote'] = true
    } else {
      messageRender = <MessageText content={content} className="message" />
    }

    return (
      <div data-message-id={message.get('id')} className="message-node">
        <div className={cx(lineClasses)} onClick={this.focusMessage}>
          <time dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')} style={timeStyle}>
            {time.format('h:mma')}
          </time>
          <span className="nick" style={{background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 85%)'}}>{message.getIn(['sender', 'name'])}</span>
          <span className="content">
            {messageRender}
            {messageEmbeds}
          </span>
        </div>
        {(children.size > 0 || entry) &&
          <div className="replies">
            {children.toSeq().map(function(nodeId) {
              return <Message key={nodeId} tree={this.props.tree} nodeId={nodeId} depth={this.props.depth + 1} displayFocusHighlight={!!entry} />
            }, this).toArray()}
            {entry && <ChatEntry />}
          </div>
        }
        <div className="reply-anchor" />
      </div>
    )
  },
})
