var _ = require('lodash')
var React = require('react')
var classNames = require('classnames')
var Reflux = require('reflux')

var FastButton = require('./fast-button')
var MessageText = require('./message-text')
var LiveTimeAgo = require('./live-time-ago')


module.exports = React.createClass({
  displayName: 'NotificationListItem',

  mixins: [
    require('./tree-node-mixin')(),
    Reflux.connect(require('../stores/clock').minute, 'now'),
  ],

  componentWillEnter: function(callback) {
    var node = this.getDOMNode()
    var height = this.getDOMNode().clientHeight
    node.style.transition = 'none'
    node.style.height = 0
    node.style.opacity = 0
    _.identity(node.offsetHeight)  // reflow so transition starts
    node.style.transition = 'all .25s ease'
    node.style.height = height + 'px'
    node.style.opacity = 1
    callback()
  },

  componentWillLeave: function(callback) {
    var node = this.getDOMNode()
    node.style.height = 0
    setTimeout(() => {
      node.style.transition = 'none'
      callback()
    }, 250)
  },

  render: function() {
    var message = this.state.node

    return (
      <FastButton component="div" className={classNames('notification', this.props.kind, {'seen': message.get('_seen')})} onClick={ev => this.props.onClick(ev, this.props.nodeId)}>
        <MessageText className="title" content={message.get('content')} maxLength={140} />
        <LiveTimeAgo className="ago" time={message.get('time')} nowText="active" />
      </FastButton>
    )
  },
})
