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
    require('./hooksmixin'),
    Reflux.connect(require('../stores/chat').store, 'chat'),
    Reflux.connect(require('../stores/focus').store, 'focus'),
    Reflux.connect(require('../stores/update').store, 'update'),
    Reflux.listenTo(actions.focusMessage, 'focusMessage'),
    Reflux.listenTo(actions.scrollToEntry, 'scrollToEntry'),
  ],

  componentWillMount: function() {
    this._lastFocusMessage = 0
  },

  onScrollbarSize: function(width) {
    this.setState({scrollbarWidth: width})
  },

  onResize: function(width) {
    this.setState({
      thin: width < 500,
    })
  },

  focusMessage: function() {
    this._lastFocusMessage = Date.now()
  },

  onScroll: function() {
    if (Date.now() - this._lastFocusMessage < 250) {
      return
    }

    var activeEl = uidocument.activeElement
    if (Heim.isTouch && this.getDOMNode().contains(activeEl) && activeEl.nodeName == 'INPUT') {
      activeEl.blur()
    }
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
    if (!uiwindow.getSelection().isCollapsed || ev.target.nodeName == 'BUTTON') {
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
        <InfoBar scrollbarWidth={this.state.scrollbarWidth} who={this.state.chat.who} roomName={this.state.chat.roomName} authType={this.state.chat.authType} updateReady={this.state.update.get('ready')} />
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
          onScroll={this.onScroll}
          onNearTop={actions.loadMoreLogs}
        >
          <div className="messages-content">
            <Messages ref="messages" />
          </div>
        </Scroller>
        {this.templateHook('page-bottom')}
      </div>
    )
  },
})
