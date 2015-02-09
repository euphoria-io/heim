var React = require('react/addons')
var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup
var cx = React.addons.classSet
var Reflux = require('reflux')

var actions = require('../actions')
var Scroller = require('./scroller')
var Messages = require('./messages')
var UserList = require('./userlist')
var NotifyToggle = require('./notifytoggle')


module.exports = React.createClass({
  displayName: 'Main',

  mixins: [
    Reflux.connect(require('../stores/chat').store, 'chat'),
    Reflux.connect(require('../stores/focus').store, 'focus'),
    Reflux.listenTo(actions.showSettings, 'showSettings'),
    Reflux.listenTo(actions.scrollToEntry, 'scrollToEntry'),
  ],

  onScrollbarSize: function(width) {
    this.setState({scrollbarWidth: width})
  },

  toggleSettings: function() {
    this.setState({settingsOpen: !this.state.settingsOpen})
  },

  showSettings: function() {
    this.setState({settingsOpen: true})
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
    if (!window.getSelection().isCollapsed) {
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
    return (
      <div className="chat">
        <Scroller
          ref="scroller"
          target=".entry"
          edgeSpace={156}
          className={cx({
            'messages-container': true,
            'focus-highlighting': !!this.state.chat.focusedMessage,
            'form-focus': this.state.focus.windowFocused && this.state.chat.connected,
          })}
          onScrollbarSize={this.onScrollbarSize}
          onNearTop={actions.loadMoreLogs}
        >
          <div className="messages-content" onMouseDownCapture={this.onMouseDown} onClickCapture={this.onClick}>
            <div className="top-right" style={{marginRight: this.state.scrollbarWidth}}>
              <div className="settings-pane">
                <ReactCSSTransitionGroup transitionName="settings">
                  {this.state.settingsOpen &&
                    <span key="content" className="settings-content">
                      <NotifyToggle />
                    </span>
                  }
                </ReactCSSTransitionGroup>
                <button type="button" className="settings" onClick={this.toggleSettings} tabIndex="-1" />
              </div>
              <UserList users={this.state.chat.who} />
            </div>
            <Messages ref="messages" />
          </div>
        </Scroller>
      </div>
    )
  },
})
