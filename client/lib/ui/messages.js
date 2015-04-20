var React = require('react/addons')
var Reflux = require('reflux')
var moment = require('moment')

var MessageList = require('./messagelist')
var ChatEntry = require('./chatentry')
var NickEntry = require('./nickentry')
var PasscodeEntry = require('./passcodeentry')


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
    if (this.state.authType == 'passcode' && this.state.authState && this.state.authState != 'trying-stored') {
      entry = <PasscodeEntry />
    } else if (this.state.joined && !this.state.nick && !this.state.tentativeNick) {
      entry = <NickEntry />
    } else if (!this.state.focusedMessage) {
      entry = <ChatEntry />
    }

    var rendered = (
      <div className="messages">
        <MessageList tree={this.state.messages} roomSettings={this.state.roomSettings} />
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
