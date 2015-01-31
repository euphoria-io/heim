var React = require('react/addons')
var Reflux = require('reflux')
var moment = require('moment')

var MessageList = require('./messagelist')
var ChatEntry = require('./chatentry')
var NickEntry = require('./nickentry')


module.exports = React.createClass({
  displayName: 'Messages',

  mixins: [
    require('react-immutable-render-mixin'),
    Reflux.connect(require('../stores/chat').store),
  ],

  render: function() {
    var now = moment()
    var disconnected = this.state.connected === false

    var entry
    if (!disconnected &&
        (!this.state.nickInFlight || !this.state.nick || this.state.nick === '') &&
        (!this.state.confirmedNick || this.state.confirmedNick === '')) {
      entry = <NickEntry />
    } else if (!this.state.focusedMessage) {
      entry = <ChatEntry />
    }

    var rendered = (
      <div className="messages">
        <MessageList tree={this.state.messages} />
        {disconnected ?
          <div key="status" className="line status disconnected">
            <time dateTime={now.toISOString()} title={now.format('MMMM Do YYYY, h:mm:ss a')}>
              {now.format('h:mma')}
            </time>
            <span className="message">reconnecting...</span>
          </div>
        : null}
        {entry}
      </div>
    )

    return rendered
  },
})
