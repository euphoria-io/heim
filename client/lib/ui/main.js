var React = require('react/addons')
var cx = React.addons.classSet
var Reflux = require('reflux')

var actions = require('../actions')
var Scroller = require('./scroller')
var Messages = require('./messages')
var ChatSidebar = require('./chatsidebar')
var ChatTopBar = require('./chattopbar')


module.exports = React.createClass({
  displayName: 'Main',

  mixins: [
    Reflux.connect(require('../stores/chat').store, 'chat'),
    Reflux.connect(require('../stores/focus').store, 'focus'),
    Reflux.listenTo(actions.scrollToEntry, 'scrollToEntry'),
  ],

  onScrollbarSize: function(width) {
    this.setState({scrollbarWidth: width})
  },

  onResize: function(width) {
    this.setState({
      thin: width < 500,
    })
  },

  scrollToEntry: function() {
    this.refs.scroller.scrollToTarget()
  },

  onMouseDown: function() {
    // FIXME: preventing/canceling a mousedown in React doesn't seem to stop
    // the subsequent click event, so we have to resort to this hack.
    this._isFocusClick = Date.now() - this.state.focus.focusChangedAt < 100
  },

  onClick: function(ev) {
    if (!window.getSelection().isCollapsed || ev.target.nodeName == 'BUTTON') {
      return
    }

    // prevent clicks to focus window and link clicks from triggering elements
    if (this._isFocusClick || ev.target.nodeName == 'A') {
      ev.stopPropagation()
    }

    if (this._isFocusClick) {
      actions.focusEntry()
    }
  },

  render: function() {
    var InfoBar = this.state.thin ? ChatTopBar : ChatSidebar
    return (
      <div className="chat" onMouseDownCapture={this.onMouseDown} onClickCapture={this.onClick}>
        {this.state.chat.authState && this.state.chat.authState != 'trying-stored' && <div className="hatch-shade fill" />}
        <Scroller
          ref="scroller"
          target=".entry"
          edgeSpace={156}
          className={cx({
            'messages-container': true,
            'focus-highlighting': !!this.state.chat.focusedMessage,
            'form-focus': this.state.focus.windowFocused && this.state.chat.connected !== false,
          })}
          onScrollbarSize={this.onScrollbarSize}
          onResize={this.onResize}
          onNearTop={actions.loadMoreLogs}
        >
          <div className="messages-content">
            <InfoBar scrollbarWidth={this.state.scrollbarWidth} who={this.state.chat.who} roomName={this.state.chat.roomName} authType={this.state.chat.authType} />
            <Messages ref="messages" />
          </div>
        </Scroller>
      </div>
    )
  },
})
