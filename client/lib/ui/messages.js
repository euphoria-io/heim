var _ = require('lodash')
var React = require('react')
var Reflux = require('reflux')
var moment = require('moment')

var actions = require('../actions')
var Message = require('./message')
var ChatEntry = require('./chatentry')
var NickEntry = require('./nickentry')


module.exports = {}

module.exports = React.createClass({
  displayName: 'Messages',

  mixins: [
    require('react-immutable-render-mixin'),
    Reflux.connect(require('../stores/chat').store),
  ],

  focusInput: function() {
    this.refs.entry.focusInput()
  },

  isFocused: function() {
    return this.refs.entry.isFocused()
  },

  render: function() {
    var now = moment()

    var entry
    if (this.state.nick) {
      entry = <ChatEntry ref="entry" nick={this.state.nick} onFormFocus={this.props.onFormFocus} />
    } else {
      entry = <NickEntry ref="entry" onFormFocus={this.props.onFormFocus} />
    }

    return (
      <div className="messages">
        {this.state.messages.map(function(message, idx) {
          return <Message key={idx} message={message} />
        }, this).toArray()}
        {entry}
        {this.state.connected == false ?
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
